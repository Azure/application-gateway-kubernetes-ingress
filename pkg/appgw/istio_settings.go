// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
)

func istioMatchDestinationIds(cbCtx *ConfigBuilderContext) ([]istioMatchIdentifier, map[istioDestinationIdentifier]interface{}) {
	matchIDs := make([]istioMatchIdentifier, 0)
	destinationIDs := make(map[istioDestinationIdentifier]interface{})
	for _, virtualService := range cbCtx.IstioVirtualServices {
		for _, rule := range virtualService.Spec.HTTP {
			destinations := make([]*v1alpha3.Destination, 0)
			for _, routeDestination := range rule.Route {
				if routeDestination.Weight != 0 {
					destinations = append(destinations, &routeDestination.Destination)
					/* TODO(rhea): Weights are being ignored for now, since this is not
					yet supported on App Gateway. Include gates from routeDestination when
					this is supported */
				}
				destinationID := generateIstioDestinationID(virtualService, &routeDestination.Destination)
				destinationIDs[destinationID] = nil
			}
			for _, match := range rule.Match {
				if match.URI == nil {
					klog.V(5).Infof("Skipped match request, no URI field. Other forms of match requests are not supported.")
					continue
				}
				matchID := generateIstioMatchID(virtualService, &rule, &match, destinations)
				matchIDs = append(matchIDs, matchID)
			}
		}
	}
	/* TODO(rhea): Filter out destinations for virtual services referencing non-existent Services */
	return matchIDs, destinationIDs
}

func (c *appGwConfigBuilder) getIstioDestinationsAndSettingsMap(cbCtx *ConfigBuilderContext) ([]n.ApplicationGatewayBackendHTTPSettings, map[istioDestinationIdentifier]*n.ApplicationGatewayBackendHTTPSettings, map[istioDestinationIdentifier]serviceBackendPortPair, error) {
	serviceBackendPairsMap := make(map[istioDestinationIdentifier]map[serviceBackendPortPair]interface{})
	backendHTTPSettingsMap := make(map[istioDestinationIdentifier]*n.ApplicationGatewayBackendHTTPSettings)
	finalServiceBackendPairMap := make(map[istioDestinationIdentifier]serviceBackendPortPair)

	var unresolvedDestinationID []istioDestinationIdentifier
	_, destinationIDs := istioMatchDestinationIds(cbCtx)
	for destinationID := range destinationIDs {
		resolvedBackendPorts := make(map[serviceBackendPortPair]interface{})

		service := c.k8sContext.GetService(destinationID.serviceKey())
		destinationPortNum := Port(destinationID.DestinationPort)
		if service == nil {
			// Once services are filtered in the istioMatchDestinationIDs function, this should never happen
			logLine := fmt.Sprintf("Unable to get the service [%s]", destinationID.serviceKey())
			klog.Errorf(logLine)
			// TODO(rhea): add error event
			pair := serviceBackendPortPair{
				ServicePort: Port(destinationPortNum),
				BackendPort: Port(destinationPortNum),
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

				// TODO(delqn): implement correctly port lookup by name
				if Port(sp.Port) == destinationPortNum || sp.TargetPort.String() == fmt.Sprint(destinationPortNum) {
					// matched a service port with a port from the service
					if sp.TargetPort.String() == "" {
						// targetPort is not defined, by default targetPort == port
						pair := serviceBackendPortPair{
							ServicePort: Port(sp.Port),
							BackendPort: Port(sp.Port),
						}
						resolvedBackendPorts[pair] = nil
					} else {
						// target port is defined as name or port number
						if sp.TargetPort.Type == intstr.Int {
							// port is defined as port number
							pair := serviceBackendPortPair{
								ServicePort: Port(sp.Port),
								BackendPort: Port(sp.TargetPort.IntVal),
							}
							resolvedBackendPorts[pair] = nil
						} else {
							// if service port is defined by name, need to resolve
							targetPortName := sp.TargetPort.StrVal
							klog.V(1).Infof("resolving port name %s", targetPortName)
							targetPortsResolved := c.resolveIstioPortName(targetPortName, &destinationID)
							for targetPort := range targetPortsResolved {
								pair := serviceBackendPortPair{
									ServicePort: Port(sp.Port),
									BackendPort: Port(targetPort),
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
			logLine := fmt.Sprintf("Unable to resolve any backend port for service [%s]", destinationID.serviceKey())
			klog.Error(logLine)
			//TODO(rhea): Add error event

			unresolvedDestinationID = append(unresolvedDestinationID, destinationID)
			break
		}

		// Merge serviceBackendPairsMap[backendID] into resolvedBackendPorts
		if _, ok := serviceBackendPairsMap[destinationID]; !ok {
			serviceBackendPairsMap[destinationID] = make(map[serviceBackendPortPair]interface{})
		}
		for portPair := range resolvedBackendPorts {
			serviceBackendPairsMap[destinationID][portPair] = nil
		}
	}
	if len(unresolvedDestinationID) > 0 {
		e := controllererrors.NewError(
			controllererrors.ErrorIstioResolvePortsForServices,
			"unable to resolve backend port for some services",
		)
		return nil, nil, nil, e
	}

	httpSettingsCollection := make(map[string]n.ApplicationGatewayBackendHTTPSettings)
	for destinationID, serviceBackendPairs := range serviceBackendPairsMap {
		if len(serviceBackendPairs) > 1 {
			// more than one possible backend port exposed through ingress
			backendServicePort := ""
			if destinationID.DestinationPort != 0 {
				backendServicePort = fmt.Sprint(destinationID.DestinationPort)
			} else {
				// TODO(delqn): implement port lookup by name
			}

			//TODO(rhea): add error event recorder
			e := controllererrors.NewErrorf(
				controllererrors.ErrorIstioMultipleServiceBackendPortBinding,
				"service:port [%s:%s] has more than one service-backend port binding",
				destinationID.serviceKey(), backendServicePort,
			)
			klog.Warning(e.Error())
			return nil, nil, nil, e
		}

		// At this point there will be only one pair
		var uniquePair serviceBackendPortPair
		for k := range serviceBackendPairs {
			uniquePair = k
		}

		finalServiceBackendPairMap[destinationID] = uniquePair
		httpSettings := c.generateIstioHTTPSettings(destinationID, uniquePair.BackendPort, cbCtx)
		httpSettingsCollection[*httpSettings.Name] = httpSettings
		backendHTTPSettingsMap[destinationID] = &httpSettings
	}

	httpSettings := make([]n.ApplicationGatewayBackendHTTPSettings, 0, len(httpSettingsCollection))
	for _, backend := range httpSettingsCollection {
		httpSettings = append(httpSettings, backend)
	}

	return httpSettings, backendHTTPSettingsMap, finalServiceBackendPairMap, nil
}

func (c *appGwConfigBuilder) generateIstioHTTPSettings(destinationID istioDestinationIdentifier, port Port, cbCtx *ConfigBuilderContext) n.ApplicationGatewayBackendHTTPSettings {
	backendServicePort := ""
	if destinationID.DestinationPort != 0 {
		backendServicePort = fmt.Sprint(destinationID.DestinationPort)
	} else {
		// TODO(delqn): Implement port lookup by name
	}
	httpSettingsName := generateHTTPSettingsName(destinationID.serviceFullName(), backendServicePort, port, destinationID.istioVirtualServiceIdentifier.Name)
	klog.V(5).Infof("Created a new HTTP setting w/ name: %s\n", httpSettingsName)
	httpSettings := n.ApplicationGatewayBackendHTTPSettings{
		Etag: to.StringPtr("*"),
		Name: &httpSettingsName,
		ID:   to.StringPtr(c.appGwIdentifier.HTTPSettingsID(httpSettingsName)),
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Protocol: n.HTTP,
			Port:     to.Int32Ptr(int32(port)),
		},
	}

	//TODO(rhea): check relevant annotations and modify http settings accordingly

	return httpSettings
}
