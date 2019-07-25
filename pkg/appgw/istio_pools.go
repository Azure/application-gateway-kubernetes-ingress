// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
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
			if destinationID.Destination.Port.Number != 0 {
				backendServicePort = string(destinationID.Destination.Port.Number)
			} else {
				backendServicePort = destinationID.Destination.Port.Name
			}
			poolName := generateAddressPoolName(destinationID.serviceFullName(), backendServicePort, serviceBackendPair.BackendPort)
			if pool, ok := addressPools[poolName]; ok {
				return pool
			}
			return newPool(poolName, subset)
		}
		logLine := fmt.Sprintf("Backend target port %d does not have matching endpoint port", serviceBackendPair.BackendPort)
		glog.Error(logLine)
		//TODO(rhea): add recorder event for error
	}
	return nil
}
