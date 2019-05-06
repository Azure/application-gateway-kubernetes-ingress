// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) BackendAddressPools(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error) {
	addressPools := make(map[string](network.ApplicationGatewayBackendAddressPool))
	emptyPool := defaultBackendAddressPool()
	addressPools[*emptyPool.Name] = emptyPool

	for backendID, serviceBackendPair := range builder.serviceBackendPairMap {
		endpoints := builder.k8sContext.GetEndpointsByService(backendID.serviceKey())
		if endpoints == nil {
			glog.Warningf("unable to get endpoints for service key [%s]", backendID.serviceKey())
			builder.backendPoolMap[backendID] = &emptyPool
			continue
		}

		for _, subset := range endpoints.Subsets {
			endpointsPortsSet := utils.NewUnorderedSet()
			for _, endpointsPort := range subset.Ports {
				if endpointsPort.Protocol != v1.ProtocolTCP {
					continue
				}
				endpointsPortsSet.Insert(endpointsPort.Port)
			}

			if endpointsPortsSet.Contains(serviceBackendPair.BackendPort) {
				addressPoolName := generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), serviceBackendPair.BackendPort)
				// The same service might be referenced in multiple ingress resources, this might result in multiple `serviceBackendPairMap` having the same service key but different
				// ingress resource. Thus, while generating the backend address pool, we should make sure that we are generating unique backend address pools.
				addressPool, ok := addressPools[addressPoolName]

				if !ok {
					addressPoolAddresses := make([](network.ApplicationGatewayBackendAddress), 0)
					for _, address := range subset.Addresses {
						ip := address.IP
						hostname := address.Hostname
						// prefer IP address
						if len(ip) != 0 {
							// address specified by ip
							addressPoolAddresses = append(addressPoolAddresses, network.ApplicationGatewayBackendAddress{IPAddress: &ip})
						} else if len(address.Hostname) != 0 {
							// address specified by hostname
							addressPoolAddresses = append(addressPoolAddresses, network.ApplicationGatewayBackendAddress{Fqdn: &hostname})
						}
					}

					addressPool = network.ApplicationGatewayBackendAddressPool{
						Etag: to.StringPtr("*"),
						Name: &addressPoolName,
						ApplicationGatewayBackendAddressPoolPropertiesFormat: &network.ApplicationGatewayBackendAddressPoolPropertiesFormat{
							BackendAddresses: &addressPoolAddresses,
						},
					}

					addressPools[*addressPool.Name] = addressPool
				}

				builder.backendPoolMap[backendID] = &addressPool
				break
			}
		}
	}

	backendPools := make([](network.ApplicationGatewayBackendAddressPool), 0)
	for _, addressPool := range addressPools {
		backendPools = append(backendPools, addressPool)
	}

	builder.appGwConfig.BackendAddressPools = &backendPools

	return builder, nil
}
