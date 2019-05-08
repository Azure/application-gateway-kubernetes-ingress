// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) HealthProbesCollection(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error) {
	backendIDs := utils.NewUnorderedSet()

	for _, ingress := range ingressList {
		defIngressBackend := ingress.Spec.Backend
		if defIngressBackend != nil {
			backendIDs.Insert(generateBackendID(ingress, nil, nil, defIngressBackend))
		}
		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			if rule.HTTP == nil {
				// skip no http rule
				continue
			}
			for pathIdx := range rule.HTTP.Paths {
				path := &rule.HTTP.Paths[pathIdx]
				backendIDs.Insert(generateBackendID(ingress, rule, path, &path.Backend))
			}
		}
	}

	healthProbeCollection := make([](network.ApplicationGatewayProbe), 0)
	defProbe := defaultProbe()
	healthProbeCollection = append(healthProbeCollection, defProbe)
	backendIDs.ForEach(func(backendInterface interface{}) {
		backendID := backendInterface.(backendIdentifier)
		ingress := backendID.Ingress
		service := builder.k8sContext.GetService(backendID.serviceKey())
		podList := builder.k8sContext.GetPodsByServiceSelector(service.Spec.Selector)
		probe := network.ApplicationGatewayProbe{
			ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{},
		}
		var protocol network.ApplicationGatewayProtocol
		var host string
		var path string
		var interval int32
		var timeout int32
		var unhealthyThreshold int32

		for _, sp := range service.Spec.Ports {
			if sp.Protocol != v1.ProtocolTCP {
				continue
			}

			for _, pod := range podList {
				for _, container := range pod.Spec.Containers {
					for _, port := range container.Ports {
						if port.ContainerPort == sp.Port {
							if container.ReadinessProbe != nil && container.ReadinessProbe.Handler.HTTPGet != nil {
								httpGet := container.ReadinessProbe.Handler.HTTPGet
								host = httpGet.Host
								path = httpGet.Path
								if httpGet.Scheme == v1.URISchemeHTTPS {
									protocol = network.HTTPS
								} else {
									protocol = network.HTTP
								}
							} else if container.LivenessProbe != nil && container.LivenessProbe.Handler.HTTPGet != nil {
								httpGet := container.LivenessProbe.Handler.HTTPGet
								host = httpGet.Host
								path = httpGet.Path
								if httpGet.Scheme == v1.URISchemeHTTPS {
									protocol = network.HTTPS
								} else {
									protocol = network.HTTP
								}
							} else if len(annotations.BackendPathPrefix(ingress)) != 0 {
								path = annotations.BackendPathPrefix(ingress)
							} else if backendID.Path != nil && len(backendID.Path.Path) != 0 {
								path = backendID.Path.Path
							}

							host = backendID.Rule.Host
						}
					}
				}
			}
		}

		name := generateProbeName(backendID.Path.Backend.ServiceName, backendID.Path.Backend.ServicePort.String(), backendID.Ingress.Name)
		probe.Name = &name
		probe.Protocol = protocol
		if len(host) != 0 {
			probe.Host = &host
		} else {
			probe.Host = defProbe.Host
		}
		if len(path) != 0 {
			probe.Path = &path
		} else {
			probe.Path = defProbe.Path
		}
		if interval != 0 {
			probe.Interval = &interval
		} else {
			probe.Interval = defProbe.Interval
		}
		if timeout != 0 {
			probe.Timeout = &timeout
		} else {
			probe.Timeout = defProbe.Timeout
		}
		if unhealthyThreshold != 0 {
			probe.UnhealthyThreshold = &unhealthyThreshold
		} else {
			probe.UnhealthyThreshold = defProbe.UnhealthyThreshold
		}

		builder.probesMap[backendID] = &probe
		healthProbeCollection = append(healthProbeCollection, probe)
	})

	builder.appGwConfig.Probes = &healthProbeCollection

	return builder, nil
}
