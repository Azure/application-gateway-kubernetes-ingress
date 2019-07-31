// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"sort"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

func (c *appGwConfigBuilder) HealthProbesCollection(cbCtx *ConfigBuilderContext) error {
	healthProbeCollection, _ := c.newProbesMap(cbCtx)
	glog.V(5).Infof("Will create %d App Gateway probes.", len(healthProbeCollection))
	agicCreatedProbes := make([]n.ApplicationGatewayProbe, 0, len(healthProbeCollection))
	for _, probe := range healthProbeCollection {
		agicCreatedProbes = append(agicCreatedProbes, probe)
	}

	if cbCtx.EnableBrownfieldDeployment {
		er := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)
		existingBlacklisted, existingNonBlacklisted := er.GetBlacklistedProbes()
		brownfield.LogProbes(existingBlacklisted, existingNonBlacklisted, agicCreatedProbes)
		agicCreatedProbes = brownfield.MergeProbes(existingBlacklisted, agicCreatedProbes)
	}

	sort.Sort(sorter.ByHealthProbeName(agicCreatedProbes))
	c.appGw.Probes = &agicCreatedProbes
	return nil
}

func (c *appGwConfigBuilder) newProbesMap(cbCtx *ConfigBuilderContext) (map[string]n.ApplicationGatewayProbe, map[backendIdentifier]*n.ApplicationGatewayProbe) {
	healthProbeCollection := make(map[string]n.ApplicationGatewayProbe)
	probesMap := make(map[backendIdentifier]*n.ApplicationGatewayProbe)
	defaultProbe := defaultProbe(c.appGwIdentifier)

	glog.V(5).Info("Adding default probe:", *defaultProbe.Name)
	healthProbeCollection[*defaultProbe.Name] = defaultProbe

	for backendID := range newBackendIdsFiltered(cbCtx) {
		probe := c.generateHealthProbe(backendID)

		if probe != nil {
			glog.V(5).Infof("Created probe %s for backend: '%s'", *probe.Name, backendID.Name)
			probesMap[backendID] = probe
			healthProbeCollection[*probe.Name] = *probe
		} else {
			glog.V(5).Infof("No k8s probe for backend: '%s'; Adding default probe: '%s'", backendID.Name, *defaultProbe.Name)
			probesMap[backendID] = &defaultProbe
		}
	}
	return healthProbeCollection, probesMap
}

func (c *appGwConfigBuilder) generateHealthProbe(backendID backendIdentifier) *n.ApplicationGatewayProbe {
	// TODO(draychev): remove GetService
	service := c.k8sContext.GetService(backendID.serviceKey())
	if service == nil {
		return nil
	}
	probe := defaultProbe(c.appGwIdentifier)
	probe.Name = to.StringPtr(generateProbeName(backendID.Path.Backend.ServiceName, backendID.Path.Backend.ServicePort.String(), backendID.Ingress))
	probe.ID = to.StringPtr(c.appGwIdentifier.probeID(*probe.Name))
	if backendID.Rule != nil && len(backendID.Rule.Host) != 0 {
		probe.Host = to.StringPtr(backendID.Rule.Host)
	}

	pathPrefix, err := annotations.BackendPathPrefix(backendID.Ingress)
	if err == nil {
		probe.Path = to.StringPtr(pathPrefix)
	} else if backendID.Path != nil && len(backendID.Path.Path) != 0 {
		probe.Path = to.StringPtr(backendID.Path.Path)
	}

	k8sProbeForServiceContainer := c.getProbeForServiceContainer(service, backendID)
	if k8sProbeForServiceContainer != nil {
		if len(k8sProbeForServiceContainer.Handler.HTTPGet.Host) != 0 {
			probe.Host = to.StringPtr(k8sProbeForServiceContainer.Handler.HTTPGet.Host)
		}
		if len(k8sProbeForServiceContainer.Handler.HTTPGet.Path) != 0 {
			probe.Path = to.StringPtr(k8sProbeForServiceContainer.Handler.HTTPGet.Path)
		}
		if k8sProbeForServiceContainer.Handler.HTTPGet.Scheme == v1.URISchemeHTTPS {
			probe.Protocol = n.HTTPS
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

	if probe.Path != nil {
		probe.Path = to.StringPtr(strings.TrimRight(*probe.Path, "*"))
	}
	return &probe
}

func (c *appGwConfigBuilder) getProbeForServiceContainer(service *v1.Service, backendID backendIdentifier) *v1.Probe {
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
				for targetPort := range c.resolvePortName(sp.Name, &backendID) {
					allPorts[targetPort] = nil
				}
			}
		}
	}

	podList := c.k8sContext.ListPodsByServiceSelector(service.Spec.Selector)
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
