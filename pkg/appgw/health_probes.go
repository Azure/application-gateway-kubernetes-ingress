// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (builder *appGwConfigBuilder) HealthProbesCollection(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error) {
	backendIDs := utils.NewUnorderedSet()
	for _, ingress := range ingressList {

		glog.Infof("[health-probes] Configuring health probes for ingress: '%s'", ingress.Name)
		if ingress.Spec.Backend != nil {
			glog.Info("[health-probes] Ingress spec has no backend. Adding a default.")
			backendIDs.Insert(generateBackendID(ingress, nil, nil, ingress.Spec.Backend))
		}

		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			glog.Infof("[health-probes] Working on ingress rule #%d: host='%s'", ruleIdx+1, rule.Host)
			if rule.HTTP == nil {
				// skip no http rule
				glog.Infof("[health-probes] Skip rule#%d for host '%s' - it has no HTTP rules.", ruleIdx+1, rule.Host)
				continue
			}
			for pathIdx := range rule.HTTP.Paths {
				path := &rule.HTTP.Paths[pathIdx]
				glog.Infof("[health-probes] Working on path #%d: '%s'", pathIdx+1, path.Path)
				backendIDs.Insert(generateBackendID(ingress, rule, path, &path.Backend))
			}
		}
	}

	defaultProbe := defaultProbe()

	healthProbeCollection := make([]network.ApplicationGatewayProbe, 0)
	glog.Infof("[health-probes] Adding default probe: '%s'", *defaultProbe.Name)
	healthProbeCollection = append(healthProbeCollection, defaultProbe)

	for _, backendIDInterface := range backendIDs.ToSlice() {
		backendID := backendIDInterface.(backendIdentifier)
		probe := builder.generateHealthProbe(backendID)

		if probe != nil {
			glog.Infof("[health-probes] Found k8s probe for backend: '%s'", backendID.Name)
			builder.probesMap[backendID] = probe
			healthProbeCollection = append(healthProbeCollection, *probe)
		} else {
			glog.Infof("[health-probes] No k8s probe for backend: '%s'; Adding default probe: '%s'", backendID.Name, *defaultProbe.Name)
			builder.probesMap[backendID] = &defaultProbe
		}
	}

	glog.Infof("[health-probes] Will create %d App Gateway probes.", len(healthProbeCollection))
	builder.appGwConfig.Probes = &healthProbeCollection
	return builder, nil
}

func (builder *appGwConfigBuilder) generateHealthProbe(backendID backendIdentifier) *network.ApplicationGatewayProbe {
	probe := defaultProbe()
	service := builder.k8sContext.GetService(backendID.serviceKey())
	if service != nil {
		probe.Name = to.StringPtr(generateProbeName(backendID.Path.Backend.ServiceName, backendID.Path.Backend.ServicePort.String(), backendID.Ingress.Name))
		if backendID.Rule != nil && len(backendID.Rule.Host) != 0 {
			probe.Host = to.StringPtr(backendID.Rule.Host)
		}

		if len(annotations.BackendPathPrefix(backendID.Ingress)) != 0 {
			probe.Path = to.StringPtr(annotations.BackendPathPrefix(backendID.Ingress))
		} else if backendID.Path != nil && len(backendID.Path.Path) != 0 {
			probe.Path = to.StringPtr(backendID.Path.Path)
		}

		k8sProbeForServiceContainer := builder.getProbeForServiceContainer(service, backendID)
		if k8sProbeForServiceContainer != nil {
			if len(k8sProbeForServiceContainer.Handler.HTTPGet.Host) != 0 {
				probe.Host = to.StringPtr(k8sProbeForServiceContainer.Handler.HTTPGet.Host)
			}
			if len(k8sProbeForServiceContainer.Handler.HTTPGet.Path) != 0 {
				probe.Path = to.StringPtr(k8sProbeForServiceContainer.Handler.HTTPGet.Path)
			}
			if k8sProbeForServiceContainer.Handler.HTTPGet.Scheme == v1.URISchemeHTTPS {
				probe.Protocol = network.HTTPS
			}
			if k8sProbeForServiceContainer.PeriodSeconds != 0 {
				probe.Interval = to.Int32Ptr(k8sProbeForServiceContainer.PeriodSeconds)
			}
			if k8sProbeForServiceContainer.TimeoutSeconds != 0 {
				probe.Timeout = to.Int32Ptr(k8sProbeForServiceContainer.TimeoutSeconds)
			}
			if k8sProbeForServiceContainer.FailureThreshold != 0 {
				probe.UnhealthyThreshold = to.Int32Ptr(k8sProbeForServiceContainer.FailureThreshold)
			}
		}
	}

	return &probe
}

func (builder *appGwConfigBuilder) getProbeForServiceContainer(service *v1.Service, backendID backendIdentifier) *v1.Probe {
	allPorts := make(map[int32]interface{})
	for _, sp := range service.Spec.Ports {
		if sp.Protocol != v1.ProtocolTCP {
			continue
		}

		if fmt.Sprint(sp.Port) == backendID.Backend.ServicePort.String() ||
			sp.Name == backendID.Backend.ServicePort.String() ||
			sp.TargetPort.String() == backendID.Backend.ServicePort.String() {

			// Matched a service port in the service
			if sp.TargetPort.String() == "" {
				allPorts[sp.Port] = nil
			} else if sp.TargetPort.Type == intstr.Int {
				// port is defined as port number
				allPorts[sp.TargetPort.IntVal] = nil
			} else {
				targetPortsResolved := builder.resolvePortName(sp.TargetPort.StrVal, &backendID)
				targetPortsResolved.ForEach(func(targetPortInterface interface{}) {
					targetPort := targetPortInterface.(int32)
					allPorts[targetPort] = nil
				})
			}
		}
	}

	podList := builder.k8sContext.GetPodsByServiceSelector(service.Spec.Selector)
	for _, pod := range podList {
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				if _, ok := allPorts[port.ContainerPort]; !ok {
					continue
				}

				var probe *v1.Probe
				if container.ReadinessProbe != nil && container.ReadinessProbe.Handler.HTTPGet != nil {
					probe = container.ReadinessProbe
				} else if container.LivenessProbe != nil && container.LivenessProbe.Handler.HTTPGet != nil {
					probe = container.LivenessProbe
				}

				return probe
			}
		}
	}

	return nil
}
