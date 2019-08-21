// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
)

func (c *appGwConfigBuilder) getIstioBackendAddressPool(destinationID istioDestinationIdentifier, serviceBackendPair serviceBackendPortPair, addressPools map[string]*n.ApplicationGatewayBackendAddressPool) *n.ApplicationGatewayBackendAddressPool {
	endpoints, err := c.k8sContext.GetEndpointsByService(destinationID.serviceKey())
	if err != nil {
		logLine := fmt.Sprintf("Failed fetching endpoints for service: %s", destinationID.serviceKey())
		glog.Errorf(logLine)
		//TODO(rhea): add recorder event for error
		return nil
	}

	for _, subset := range endpoints.Subsets {
		if _, portExists := getUniqueTCPPorts(subset)[serviceBackendPair.BackendPort]; portExists {
			backendServicePort := ""
			if destinationID.DestinationPort != 0 {
				backendServicePort = fmt.Sprint(destinationID.DestinationPort)
			} else {
				// TODO(delqn): lookup port by name
			}
			poolName := generateAddressPoolName(destinationID.serviceFullName(), backendServicePort, serviceBackendPair.BackendPort)
			if pool, ok := addressPools[poolName]; ok {
				return pool
			}
			pool := c.newPool(poolName, subset)
			pool.ID = to.StringPtr(c.appGwIdentifier.addressPoolID(poolName))
			return pool
		}
		logLine := fmt.Sprintf("Backend target port %d does not have matching endpoint port", serviceBackendPair.BackendPort)
		glog.Error(logLine)
		//TODO(rhea): add recorder event for error
	}
	return nil
}

func (c *appGwConfigBuilder) newIstioBackendPoolMap(cbCtx *ConfigBuilderContext) map[istioDestinationIdentifier]*n.ApplicationGatewayBackendAddressPool {
	defaultPool := defaultBackendAddressPool(c.appGwIdentifier)
	addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
		*defaultPool.Name: &defaultPool,
	}
	backendPoolMap := make(map[istioDestinationIdentifier]*n.ApplicationGatewayBackendAddressPool)
	_, _, istioServiceBackendPairMap, _ := c.getIstioDestinationsAndSettingsMap(cbCtx)
	for destinationID, serviceBackendPair := range istioServiceBackendPairMap {
		backendPoolMap[destinationID] = &defaultPool
		if pool := c.getIstioBackendAddressPool(destinationID, serviceBackendPair, addressPools); pool != nil {
			backendPoolMap[destinationID] = pool
		}
	}
	return backendPoolMap
}
