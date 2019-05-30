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

// A facility to sort slices of ApplicationGatewayBackendAddress by IP, FQDN
type byIPFQDN []n.ApplicationGatewayBackendAddress

func (a byIPFQDN) Len() int      { return len(a) }
func (a byIPFQDN) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byIPFQDN) Less(i, j int) bool {
	if a[i].IPAddress != nil && a[j].IPAddress != nil && len(*a[i].IPAddress) > 0 {
		return *a[i].IPAddress < *a[j].IPAddress
	} else if a[i].Fqdn != nil && a[j].Fqdn != nil && len(*a[i].Fqdn) != 0 {
		return *a[i].Fqdn < *a[j].Fqdn
	}
	return false
}

func getAddresses(subset v1.EndpointSubset) *[]n.ApplicationGatewayBackendAddress {
	addrSet := make(map[n.ApplicationGatewayBackendAddress]interface{})
	for _, address := range subset.Addresses {
		// prefer IP address
		if len(address.IP) != 0 {
			// address specified by ip
			addrSet[n.ApplicationGatewayBackendAddress{IPAddress: to.StringPtr(address.IP)}] = nil
		} else if len(address.Hostname) != 0 {
			// address specified by hostname
			addrSet[n.ApplicationGatewayBackendAddress{Fqdn: to.StringPtr(address.Hostname)}] = nil
		}
	}
	var addresses []n.ApplicationGatewayBackendAddress
	for addr := range addrSet {
		addresses = append(addresses, addr)
	}
	sort.Sort(byIPFQDN(addresses))
	return &addresses
}

func (builder *appGwConfigBuilder) BackendAddressPools(ingressList []*v1beta1.Ingress) (ConfigBuilder, error) {
	defaultPool := defaultBackendAddressPool()
	addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
		*defaultPool.Name: &defaultPool,
	}
	for backendID, serviceBackendPair := range builder.getServiceBackendPairMap() {
		if pool := builder.getBackendAddressPool(backendID, serviceBackendPair, &defaultPool, &addressPools); pool != nil {
			// TODO(draychev): deprecate the caching of state in builder.backendPoolMap
			builder.backendPoolMap[backendID] = pool
		}
	}

	addressPool := make([]n.ApplicationGatewayBackendAddressPool, 0)
	for _, addr := range addressPools {
		addressPool = append(addressPool, *addr)
	}
	builder.appGwConfig.BackendAddressPools = &addressPool
	return builder, nil
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

func (builder *appGwConfigBuilder) getBackendAddressPool(backendID backendIdentifier, serviceBackendPair serviceBackendPortPair, defaultPool *n.ApplicationGatewayBackendAddressPool, addressPools *map[string]*n.ApplicationGatewayBackendAddressPool) *n.ApplicationGatewayBackendAddressPool {
	endpoints := builder.k8sContext.GetEndpointsByService(backendID.serviceKey())
	if endpoints == nil {
		logLine := fmt.Sprintf("Unable to get endpoints for service key [%s]", backendID.serviceKey())
		builder.recorder.Event(backendID.Ingress, v1.EventTypeWarning, "EndpointsEmpty", logLine)
		glog.Warning(logLine)
		return defaultPool
	}

	for _, subset := range endpoints.Subsets {
		if _, portExists := getPorts(subset)[serviceBackendPair.BackendPort]; portExists {
			poolName := generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), serviceBackendPair.BackendPort)
			// The same service might be referenced in multiple ingress resources, this might result in multiple `serviceBackendPairMap` having the same service key but different
			// ingress resource. Thus, while generating the backend address pool, we should make sure that we are generating unique backend address pools.
			pool, ok := (*addressPools)[poolName]
			if !ok {
				// Make a new Backend Address Pool
				pool = newPool(poolName, subset)
				(*addressPools)[poolName] = pool
			}
			// TODO(draychev): deprecate the caching of state in builder.backendPoolMap
			builder.backendPoolMap[backendID] = pool
			break
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
			BackendAddresses: getAddresses(subset),
		},
	}
}
