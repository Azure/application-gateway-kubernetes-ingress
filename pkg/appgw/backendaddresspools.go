// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) BackendAddressPools(ingressList []*v1beta1.Ingress) (ConfigBuilder, error) {
	defaultPool := defaultBackendAddressPool()
	addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
		*defaultPool.Name: defaultPool,
	}
	for backendID, serviceBackendPair := range builder.getServiceBackendPairMap() {
		if pool := builder.getBackendAddressPool(backendID, serviceBackendPair, addressPools); pool != nil {
			// TODO(draychev): deprecate the caching of state in builder.backendPoolMap
			builder.backendPoolMap[backendID] = pool
			addressPools[*pool.Name] = pool
		}
	}
	builder.appGwConfig.BackendAddressPools = getBackendPoolMapValues(&addressPools)
	return builder, nil
}

func getBackendPoolMapValues(m *map[string]*n.ApplicationGatewayBackendAddressPool) *[]n.ApplicationGatewayBackendAddressPool {
	var backendAddressPools []n.ApplicationGatewayBackendAddressPool
	for _, addr := range *m {
		backendAddressPools = append(backendAddressPools, *addr)
	}
	return &backendAddressPools
}

func getBackendAddressMapKeys(m *map[n.ApplicationGatewayBackendAddress]interface{}) *[]n.ApplicationGatewayBackendAddress {
	var addresses []n.ApplicationGatewayBackendAddress
	for addr := range *m {
		addresses = append(addresses, addr)
	}
	sort.Sort(byIPFQDN(addresses))
	return &addresses
}

func getPorts(subset v1.EndpointSubset) map[int32]interface{} {
	ports := make(map[int32]interface{})
	for _, endpointsPort := range subset.Ports {
		if endpointsPort.Protocol == v1.ProtocolTCP {
			ports[endpointsPort.Port] = nil
		}
	}
	return ports
}

func (builder *appGwConfigBuilder) getBackendAddressPool(backendID backendIdentifier, serviceBackendPair serviceBackendPortPair, addressPools map[string]*n.ApplicationGatewayBackendAddressPool) *n.ApplicationGatewayBackendAddressPool {
	endpoints := builder.k8sContext.GetEndpointsByService(backendID.serviceKey())
	if endpoints == nil {
		logLine := fmt.Sprintf("Unable to get endpoints for service key [%s]", backendID.serviceKey())
		builder.recorder.Event(backendID.Ingress, v1.EventTypeWarning, "EndpointsEmpty", logLine)
		glog.Warning(logLine)
		return defaultBackendAddressPool()
	}

	for _, subset := range endpoints.Subsets {
		if _, portExists := getPorts(subset)[serviceBackendPair.BackendPort]; portExists {
			poolName := generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), serviceBackendPair.BackendPort)
			// The same service might be referenced in multiple ingress resources, this might result in multiple `serviceBackendPairMap` having the same service key but different
			// ingress resource. Thus, while generating the backend address pool, we should make sure that we are generating unique backend address pools.
			if pool, ok := addressPools[poolName]; ok {
				return pool
			}
			return newPool(poolName, subset)
		}
	}
	return nil
}

func (builder *appGwConfigBuilder) getServiceBackendPairMap() map[backendIdentifier]serviceBackendPortPair {
	// TODO(draychev): deprecate the use of builder.serviceBackendPairMap
	// Create this struct here instead of backendhttpsettings.go
	return builder.serviceBackendPairMap
}

func newPool(poolName string, subset v1.EndpointSubset) *n.ApplicationGatewayBackendAddressPool {
	return &n.ApplicationGatewayBackendAddressPool{
		Etag: to.StringPtr("*"),
		Name: &poolName,
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: getAddressesForSubset(subset),
		},
	}
}

func getAddressesForSubset(subset v1.EndpointSubset) *[]n.ApplicationGatewayBackendAddress {
	// We make separate maps for IP and FQDN to ensure uniqueness within the 2 groups
	// We cannot use ApplicationGatewayBackendAddress as it contains pointer to strings and the same IP string
	// at a different address would be 2 unique keys.
	addrSet := make(map[n.ApplicationGatewayBackendAddress]interface{})
	ips := make(map[string]interface{})
	fqdns := make(map[string]interface{})
	for _, address := range subset.Addresses {
		// prefer IP address
		if len(address.IP) != 0 {
			// address specified by ip
			ips[address.IP] = nil
		} else if len(address.Hostname) != 0 {
			// address specified by hostname
			fqdns[address.Hostname] = nil
		}
	}

	for ip := range ips {
		addrSet[n.ApplicationGatewayBackendAddress{IPAddress: to.StringPtr(ip)}] = nil
	}
	for fqdn := range fqdns {
		addrSet[n.ApplicationGatewayBackendAddress{IPAddress: to.StringPtr(fqdn)}] = nil
	}
	return getBackendAddressMapKeys(&addrSet)
}
