// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) BackendAddressPools(ingressList []*v1beta1.Ingress) (ConfigBuilder, error) {
	backendPools := make([]n.ApplicationGatewayBackendAddressPool, 0)
	for _, pool := range builder.getPools() {
		poolJSON, _ := pool.MarshalJSON()
		glog.Info("Appending pool", string(poolJSON))
		backendPools = append(backendPools, pool)
	}
	builder.appGwConfig.BackendAddressPools = &backendPools
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

func (builder *appGwConfigBuilder) getPools() []n.ApplicationGatewayBackendAddressPool {
	defaultPool := defaultBackendAddressPool()
	addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
		*defaultPool.Name: &defaultPool,
	}
	defaultPoolJSON, _ := defaultPool.MarshalJSON()
	glog.Info("Added default backend pool:", string(defaultPoolJSON))

	for backendID, serviceBackendPair := range builder.getServiceBackendPairMap() {
		endpoints := builder.k8sContext.GetEndpointsByService(backendID.serviceKey())
		if endpoints == nil {

			logLine := fmt.Sprintf("Unable to get endpoints for service key [%s]", backendID.serviceKey())
			builder.recorder.Event(backendID.Ingress, v1.EventTypeWarning, "EndpointsEmpty", logLine)
			glog.Warning(logLine)

			// TODO(draychev): deprecate the caching of state in builder.backendPoolMap
			builder.backendPoolMap[backendID] = &defaultPool
			continue
		}

		for _, subset := range endpoints.Subsets {
			port := serviceBackendPair.BackendPort
			if _, portExists := getPorts(subset)[port]; portExists {
				addressPoolName := generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), port)
				// The same service might be referenced in multiple ingress resources, this might result in multiple `serviceBackendPairMap` having the same service key but different
				// ingress resource. Thus, while generating the backend address pool, we should make sure that we are generating unique backend address pools.
				if pool, ok := addressPools[addressPoolName]; ok {
					// TODO(draychev): deprecate the caching of state in builder.backendPoolMap
					builder.backendPoolMap[backendID] = pool
					break
				} else {
					addressPools[addressPoolName] = newPool(&addressPoolName, getAddresses(subset))
				}
			}
		}
	}

	a := make([]n.ApplicationGatewayBackendAddressPool, len(addressPools))
	for _, addr := range addressPools {
		a = append(a, *addr)
	}
	return a
}

func getAddresses(subset v1.EndpointSubset) *[]n.ApplicationGatewayBackendAddress {
	// Use separate buckets for IPs and FQDNs; The ApplicationGatewayBackendAddress struct here is not usable as map key
	ipSet := make(map[string]interface{})
	fqdnSet := make(map[string]interface{})
	addrSet := make(map[n.ApplicationGatewayBackendAddress]interface{})

	for _, address := range subset.Addresses {
		// prefer IP address
		if len(address.IP) != 0 {
			glog.Infof(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> %s", address.IP)
			// address specified by ip
			ipSet[address.IP] = nil
			addrSet[n.ApplicationGatewayBackendAddress{IPAddress: to.StringPtr(address.IP)}] = nil
		} else if len(address.Hostname) != 0 {
			// address specified by hostname
			fqdnSet[address.Hostname] = nil
			addrSet[n.ApplicationGatewayBackendAddress{Fqdn: to.StringPtr(address.Hostname)}] = nil
		}
	}
	glog.Infof("ips     >> %+v", ipSet)
	glog.Infof("fqdns   >> %+v", fqdnSet)
	glog.Infof("addrSet >> %+v", addrSet)
	addresses := make([]n.ApplicationGatewayBackendAddress, 0)
	/*
	for ip := range ipSet {
		addresses = append(addresses, n.ApplicationGatewayBackendAddress{IPAddress: to.StringPtr(ip)})
	}
	for fqdn := range fqdnSet {
		addresses = append(addresses, n.ApplicationGatewayBackendAddress{Fqdn: to.StringPtr(fqdn)})
	}
*/
	for x := range addrSet {
		addresses = append(addresses, x)
	}
	glog.Infof(">>>> %+v", addresses)
	return &addresses
}

func (builder *appGwConfigBuilder) getServiceBackendPairMap() map[backendIdentifier]serviceBackendPortPair {
	// TODO(draychev): deprecate the use of builder.serviceBackendPairMap
	// Create this struct here instead of backendhttpsettings.go
	return builder.serviceBackendPairMap
}

func newPool(addressPoolName *string, addressPoolAddresses *[]n.ApplicationGatewayBackendAddress) *n.ApplicationGatewayBackendAddressPool {
	return &n.ApplicationGatewayBackendAddressPool{
		Etag: to.StringPtr("*"),
		Name: addressPoolName,
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: addressPoolAddresses,
		},
	}
}
