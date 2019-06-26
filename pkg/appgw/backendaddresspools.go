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

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

func (c *appGwConfigBuilder) BackendAddressPools(cbCtx *ConfigBuilderContext) error {
	defaultPool := defaultBackendAddressPool()
	addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
		*defaultPool.Name: defaultPool,
	}
	_, _, serviceBackendPairMap, _ := c.getBackendsAndSettingsMap(cbCtx.IngressList, cbCtx.ServiceList)
	for backendID, serviceBackendPair := range serviceBackendPairMap {
		glog.V(5).Info("Constructing backend pool for service:", backendID.serviceKey())
		if pool := c.getBackendAddressPool(backendID, serviceBackendPair, addressPools); pool != nil {
			addressPools[*pool.Name] = pool
		}
	}
	pools := getBackendPoolMapValues(&addressPools)
	if pools != nil {
		sort.Sort(sorter.ByBackendPoolName(*pools))
	}
	c.appGw.BackendAddressPools = pools
	return nil
}

func (c *appGwConfigBuilder) newBackendPoolMap(ingressList []*v1beta1.Ingress, serviceList []*v1.Service) map[backendIdentifier]*n.ApplicationGatewayBackendAddressPool {
	defaultPool := defaultBackendAddressPool()
	addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
		*defaultPool.Name: defaultPool,
	}
	backendPoolMap := make(map[backendIdentifier]*n.ApplicationGatewayBackendAddressPool)
	_, _, serviceBackendPairMap, _ := c.getBackendsAndSettingsMap(ingressList, serviceList)
	for backendID, serviceBackendPair := range serviceBackendPairMap {
		backendPoolMap[backendID] = defaultPool
		if pool := c.getBackendAddressPool(backendID, serviceBackendPair, addressPools); pool != nil {
			backendPoolMap[backendID] = pool
		}
	}
	return backendPoolMap
}

func getBackendPoolMapValues(m *map[string]*n.ApplicationGatewayBackendAddressPool) *[]n.ApplicationGatewayBackendAddressPool {
	var backendAddressPools []n.ApplicationGatewayBackendAddressPool
	for _, addr := range *m {
		backendAddressPools = append(backendAddressPools, *addr)
	}
	return &backendAddressPools
}

func (c *appGwConfigBuilder) getBackendAddressPool(backendID backendIdentifier, serviceBackendPair serviceBackendPortPair, addressPools map[string]*n.ApplicationGatewayBackendAddressPool) *n.ApplicationGatewayBackendAddressPool {
	endpoints, err := c.k8sContext.GetEndpointsByService(backendID.serviceKey())
	if err != nil {
		logLine := fmt.Sprintf("Failed fetching endpoints for service: %s", backendID.serviceKey())
		glog.Errorf(logLine)
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonEndpointsEmpty, logLine)
		return nil
	}

	for _, subset := range endpoints.Subsets {
		if _, portExists := getUniqueTCPPorts(subset)[serviceBackendPair.BackendPort]; portExists {
			poolName := generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), serviceBackendPair.BackendPort)
			// The same service might be referenced in multiple ingress resources, this might result in multiple `serviceBackendPairMap` having the same service key but different
			// ingress resource. Thus, while generating the backend address pool, we should make sure that we are generating unique backend address pools.
			if pool, ok := addressPools[poolName]; ok {
				return pool
			}
			return newPool(poolName, subset)
		}
		logLine := fmt.Sprintf("Backend target port %d does not have matching endpoint port", serviceBackendPair.BackendPort)
		glog.Error(logLine)
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonBackendPortTargetMatch, logLine)
	}
	return nil
}

func getUniqueTCPPorts(subset v1.EndpointSubset) map[int32]interface{} {
	ports := make(map[int32]interface{})
	for _, endpointsPort := range subset.Ports {
		if endpointsPort.Protocol == v1.ProtocolTCP {
			ports[endpointsPort.Port] = nil
		}
	}
	return ports
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
		addrSet[n.ApplicationGatewayBackendAddress{Fqdn: to.StringPtr(fqdn)}] = nil
	}
	return getBackendAddressMapKeys(&addrSet)
}

func getBackendAddressMapKeys(m *map[n.ApplicationGatewayBackendAddress]interface{}) *[]n.ApplicationGatewayBackendAddress {
	var addresses []n.ApplicationGatewayBackendAddress
	for addr := range *m {
		addresses = append(addresses, addr)
	}
	sort.Sort(sorter.ByIPFQDN(addresses))
	return &addresses
}
