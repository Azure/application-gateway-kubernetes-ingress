// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"errors"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) BackendAddressPools(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error) {
	addressPools := make([](network.ApplicationGatewayBackendAddressPool), 0)
	addressPools = append(addressPools, defaultBackendAddressPool())

	for backendID, serviceBackendPair := range builder.serviceBackendPairMap {
		endpoints := builder.k8sContext.GetEndpointsByService(backendID.serviceKey())
		if endpoints == nil {
			glog.Warningf("unable to get endpoints for service key [%s]", backendID.serviceKey())
			return builder, errors.New("unable to get endpoints for service")
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
				addressPoolName := generateAddressPoolName(backendID.serviceFullName(), backendID.ServicePort.String(), serviceBackendPair.BackendPort)
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
				addressPool := network.ApplicationGatewayBackendAddressPool{
					Etag: to.StringPtr("*"),
					Name: &addressPoolName,
					ApplicationGatewayBackendAddressPoolPropertiesFormat: &network.ApplicationGatewayBackendAddressPoolPropertiesFormat{
						BackendAddresses: &addressPoolAddresses,
					},
				}
				addressPools = append(addressPools, addressPool)
				builder.backendPoolMap[backendID] = &addressPool
				break
			}
		}
	}

	builder.appGwConfig.BackendAddressPools = &addressPools

	return builder, nil
}
