// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"errors"
	"fmt"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (builder *appGwConfigBuilder) BackendHTTPSettingsCollection(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error) {
	backendIDs := utils.NewUnorderedSet()
	serviceBackendPairsMap := make(map[backendIdentifier](utils.UnorderedSet))

	// find all ServiceName:ServicePort pairs from the ingress list
	for _, ingress := range ingressList {
		defIngressBackend := ingress.Spec.Backend
		if defIngressBackend != nil {
			backendIDs.Insert(generateBackendID(ingress, defIngressBackend))
		}
		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				// skip no http rule
				continue
			}
			for _, path := range rule.HTTP.Paths {
				backendIDs.Insert(generateBackendID(ingress, &path.Backend))
			}
		}
	}

	unresolvedBackendID := make([]backendIdentifier, 0)
	backendIDs.ForEach(func(backendIDInterface interface{}) {
		backendID := backendIDInterface.(backendIdentifier)

		service := builder.k8sContext.GetService(backendID.serviceKey())
		if service == nil {
			glog.V(1).Infof("unable to get the service [%s]", backendID.serviceKey())
			unresolvedBackendID = append(unresolvedBackendID, backendID)
			return
		}

		// find the backend port number
		resolvedBackendPorts := utils.NewUnorderedSet()
		for _, sp := range service.Spec.Ports {
			// check if any service ports matches the specified ports
			if sp.Protocol != v1.ProtocolTCP {
				// ignore UDP ports
				continue
			}
			if fmt.Sprint(sp.Port) == backendID.ServicePort.String() ||
				sp.Name == backendID.ServicePort.String() ||
				sp.TargetPort.String() == backendID.ServicePort.String() {
				// matched a service port with a port from the service

				if sp.TargetPort.String() == "" {
					// targetPort is not defined, by default targetPort == port
					resolvedBackendPorts.Insert(serviceBackendPortPair{
						ServicePort: sp.Port,
						BackendPort: sp.Port,
					})
				} else {
					// target port is defined as name or port number
					if sp.TargetPort.Type == intstr.Int {
						// port is defined as port number
						resolvedBackendPorts.Insert(serviceBackendPortPair{
							ServicePort: sp.Port,
							BackendPort: sp.TargetPort.IntVal,
						})
					} else {
						// if service port is defined by name, need to resolve
						targetPortName := sp.TargetPort.StrVal
						glog.V(1).Infof("resolving port name %s", targetPortName)
						targetPortsResolved := builder.resolvePortName(targetPortName, &backendID)
						targetPortsResolved.ForEach(func(targetPortInterface interface{}) {
							targetPort := targetPortInterface.(int32)
							resolvedBackendPorts.Insert(serviceBackendPortPair{
								ServicePort: sp.Port,
								BackendPort: targetPort,
							})
						})
					}
				}
				break
			}
		}

		if resolvedBackendPorts.Size() == 0 {
			glog.V(1).Infof("unable to resolve any backend port for service [%s]", backendID.serviceKey())
			unresolvedBackendID = append(unresolvedBackendID, backendID)
			return
		}

		if serviceBackendPairsMap[backendID] == nil {
			serviceBackendPairsMap[backendID] = utils.NewUnorderedSet()
		}
		serviceBackendPairsMap[backendID] = serviceBackendPairsMap[backendID].Union(resolvedBackendPorts)
	})

	if len(unresolvedBackendID) > 0 {
		return builder, errors.New("unable to resolve backend port for some services")
	}

	httpSettingsCollection := make([](network.ApplicationGatewayBackendHTTPSettings), 0)
	httpSettingsCollection = append(httpSettingsCollection, defaultBackendHTTPSettings())

	// enforce single pair relationship between service port and backend port
	for backendID, serviceBackendPairs := range serviceBackendPairsMap {
		if serviceBackendPairs.Size() > 1 {
			// more than one possible backend port exposed through ingress
			glog.Warningf("service:port [%s:%s] has more than one service-backend port binding",
				backendID.serviceKey(), backendID.ServicePort.String())
			return builder, errors.New("more than one service-backend port binding is not allowed")
		}
		var uniquePair serviceBackendPortPair
		serviceBackendPairs.ForEach(func(pairI interface{}) {
			uniquePair = pairI.(serviceBackendPortPair)
		})

		builder.serviceBackendPairMap[backendID] = uniquePair

		httpSettingsName := generateHTTPSettingsName(backendID.serviceFullName(), backendID.ServicePort.String(), uniquePair.BackendPort)
		httpSettingsPort := uniquePair.BackendPort
		httpSettings := network.ApplicationGatewayBackendHTTPSettings{
			Etag: to.StringPtr("*"),
			Name: &httpSettingsName,
			ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
				Protocol: network.HTTP,
				Port:     &httpSettingsPort,
			},
		}
		// other settings should come from annotations
		httpSettingsCollection = append(httpSettingsCollection, httpSettings)
		builder.backendHTTPSettingsMap[backendID] = &httpSettings
	}

	builder.appGwConfig.BackendHTTPSettingsCollection = &httpSettingsCollection

	return builder, nil
}
