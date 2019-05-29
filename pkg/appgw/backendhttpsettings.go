// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"errors"
	"fmt"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (builder *appGwConfigBuilder) BackendHTTPSettingsCollection(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error) {
	backendIDs := make(map[backendIdentifier]interface{})
	serviceBackendPairsMap := make(map[backendIdentifier]map[serviceBackendPortPair]interface{})

	for _, ingress := range ingressList {
		defIngressBackend := ingress.Spec.Backend
		if defIngressBackend != nil {
			backendIDs[generateBackendID(ingress, nil, nil, defIngressBackend)] = nil
		}
		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			if rule.HTTP == nil {
				// skip no http rule
				continue
			}
			for pathIdx := range rule.HTTP.Paths {
				path := &rule.HTTP.Paths[pathIdx]
				backendIDs[generateBackendID(ingress, rule, path, &path.Backend)] = nil
			}
		}
	}

	unresolvedBackendID := make([]backendIdentifier, 0)
	for backendID := range backendIDs {
		resolvedBackendPorts := make(map[serviceBackendPortPair]interface{})

		service := builder.k8sContext.GetService(backendID.serviceKey())
		if service == nil {
			logLine := fmt.Sprintf("Unable to get the service [%s]", backendID.serviceKey())
			builder.recorder.Event(backendID.Ingress, v1.EventTypeWarning, "ServiceNotFound", logLine)
			glog.Errorf(logLine)
			pair := serviceBackendPortPair{
				ServicePort: backendID.Backend.ServicePort.IntVal,
				BackendPort: backendID.Backend.ServicePort.IntVal,
			}
			resolvedBackendPorts[pair] = nil
		} else {
			for _, sp := range service.Spec.Ports {
				// find the backend port number
				// check if any service ports matches the specified ports
				if sp.Protocol != v1.ProtocolTCP {
					// ignore UDP ports
					continue
				}
				if fmt.Sprint(sp.Port) == backendID.Backend.ServicePort.String() ||
					sp.Name == backendID.Backend.ServicePort.String() ||
					sp.TargetPort.String() == backendID.Backend.ServicePort.String() {
					// matched a service port with a port from the service

					if sp.TargetPort.String() == "" {
						// targetPort is not defined, by default targetPort == port
						pair := serviceBackendPortPair{
							ServicePort: sp.Port,
							BackendPort: sp.Port,
						}
						resolvedBackendPorts[pair] = nil
					} else {
						// target port is defined as name or port number
						if sp.TargetPort.Type == intstr.Int {
							// port is defined as port number
							pair := serviceBackendPortPair{
								ServicePort: sp.Port,
								BackendPort: sp.TargetPort.IntVal,
							}
							resolvedBackendPorts[pair] = nil
						} else {
							// if service port is defined by name, need to resolve
							targetPortName := sp.TargetPort.StrVal
							glog.V(1).Infof("resolving port name %s", targetPortName)
							targetPortsResolved := builder.resolvePortName(targetPortName, &backendID)
							for targetPort := range targetPortsResolved {
								pair := serviceBackendPortPair{
									ServicePort: sp.Port,
									BackendPort: targetPort,
								}
								resolvedBackendPorts[pair] = nil
							}
						}
					}
					break
				}
			}
		}

		if len(resolvedBackendPorts) == 0 {
			logLine := fmt.Sprintf("Unable to resolve any backend port for service [%s]", backendID.serviceKey())
			builder.recorder.Event(backendID.Ingress, v1.EventTypeWarning, "PortResolutionError", logLine)
			glog.Error(logLine)

			unresolvedBackendID = append(unresolvedBackendID, backendID)
			break
		}

		if serviceBackendPairsMap[backendID] == nil {
			serviceBackendPairsMap[backendID] = make(map[serviceBackendPortPair]interface{})
		}

		// Merge serviceBackendPairsMap[backendID] into resolvedBackendPorts
		if _, ok := serviceBackendPairsMap[backendID]; !ok {
			serviceBackendPairsMap[backendID] = make(map[serviceBackendPortPair]interface{})
		}
		for beID := range resolvedBackendPorts {
			serviceBackendPairsMap[backendID][beID] = nil
		}
	}

	if len(unresolvedBackendID) > 0 {
		return builder, errors.New("unable to resolve backend port for some services")
	}

	probeID := builder.appGwIdentifier.probeID(defaultProbeName)
	httpSettingsCollection := make(map[string]network.ApplicationGatewayBackendHTTPSettings)
	defaultBackend := defaultBackendHTTPSettings(probeID)
	httpSettingsCollection[*defaultBackend.Name] = defaultBackend

	// enforce single pair relationship between service port and backend port
	for backendID, serviceBackendPairs := range serviceBackendPairsMap {
		if len(serviceBackendPairs) > 1 {
			// more than one possible backend port exposed through ingress
			logLine := fmt.Sprintf("service:port [%s:%s] has more than one service-backend port binding",
				backendID.serviceKey(), backendID.Backend.ServicePort.String())
			builder.recorder.Event(backendID.Ingress, v1.EventTypeWarning, "PortResolutionError", logLine)
			glog.Warning(logLine)
			return builder, errors.New("more than one service-backend port binding is not allowed")
		}

		// At this point there will be only one pair
		var uniquePair serviceBackendPortPair
		for k := range serviceBackendPairs {
			uniquePair = k
		}

		builder.serviceBackendPairMap[backendID] = uniquePair

		probeName := builder.probesMap[backendID].Name
		probeID := builder.appGwIdentifier.probeID(*probeName)
		httpSettingsName := generateHTTPSettingsName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), uniquePair.BackendPort, backendID.Ingress.Name)
		glog.Infof("Created a new HTTP setting w/ name: %s\n", httpSettingsName)
		httpSettingsPort := uniquePair.BackendPort
		backendPathPrefix := to.StringPtr(annotations.BackendPathPrefix(backendID.Ingress))
		httpSettings := network.ApplicationGatewayBackendHTTPSettings{
			Etag: to.StringPtr("*"),
			Name: &httpSettingsName,
			ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
				Protocol: network.HTTP,
				Port:     &httpSettingsPort,
				Path:     backendPathPrefix,
				Probe:    resourceRef(probeID),
			},
		}
		// other settings should come from annotations
		httpSettingsCollection[*httpSettings.Name] = httpSettings
		builder.backendHTTPSettingsMap[backendID] = &httpSettings
	}

	backends := make([]network.ApplicationGatewayBackendHTTPSettings, 0)
	for _, backend := range httpSettingsCollection {
		backends = append(backends, backend)
	}

	builder.appGwConfig.BackendHTTPSettingsCollection = &backends

	return builder, nil
}
