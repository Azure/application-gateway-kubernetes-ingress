// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"encoding/json"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// GetManagedPools filters the given list of backend pools to the list pools that AGIC is allowed to manage.
func GetManagedPools(pools []n.ApplicationGatewayBackendAddressPool, prohibited []*ptv1.AzureIngressProhibitedTarget, ctx PoolContext) []n.ApplicationGatewayBackendAddressPool {
	blacklist := GetTargetBlacklist(prohibited)
	if len(*blacklist) == 0 {
		// There is no TargetBlacklist - AGIC will manage all configuration.
		return pools
	}

	poolToTarget := ctx.getPoolToTargetsMap()

	// Figure out if the given BackendAddressPool is blacklisted. It will be if it has a host/path that
	// has been referenced in a AzureIngressProhibitedTarget CRD (even if it has some other paths that are not)
	isPoolBlacklisted := func(pool n.ApplicationGatewayBackendAddressPool) bool {
		targetsForPool := poolToTarget[backendPoolName(*pool.Name)]
		for _, target := range targetsForPool {
			if target.IsBlacklisted(blacklist) {
				logTarget(5, target, "in blacklist")
				return true
			}
			logTarget(5, target, "implicitly managed")
		}
		return false
	}

	var managedPools []n.ApplicationGatewayBackendAddressPool
	for _, pool := range pools {
		if isPoolBlacklisted(pool) {
			glog.V(5).Infof("Backend Address Pool %s is blacklisted", *pool.Name)
			continue
		}
		glog.V(5).Infof("Backend Address Pool %s is NOT blacklisted", *pool.Name)
		managedPools = append(managedPools, pool)
	}
	return managedPools
}

// PruneManagedPools removes the managed pools from the given list of pools; resulting in a list of pools not managed by AGIC.
func PruneManagedPools(pools []n.ApplicationGatewayBackendAddressPool, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget, ctx PoolContext) []n.ApplicationGatewayBackendAddressPool {
	managedPools := GetManagedPools(pools, prohibitedTargets, ctx)
	if managedPools == nil {
		return pools
	}
	managedByName := indexByName(managedPools)
	var unmanagedPools []n.ApplicationGatewayBackendAddressPool
	for _, pool := range pools {
		if _, isManaged := managedByName[backendPoolName(*pool.Name)]; isManaged {
			continue
		}
		unmanagedPools = append(unmanagedPools, pool)
	}
	return unmanagedPools
}

// MergePools merges list of lists of backend address pools into a single list, maintaining uniqueness.
func MergePools(pools ...[]n.ApplicationGatewayBackendAddressPool) []n.ApplicationGatewayBackendAddressPool {
	uniqPool := make(poolsByName)
	for _, bucket := range pools {
		for _, pool := range bucket {
			uniqPool[backendPoolName(*pool.Name)] = pool
		}
	}
	var merged []n.ApplicationGatewayBackendAddressPool
	for _, pool := range uniqPool {
		merged = append(merged, pool)
	}
	return merged
}

func indexByName(pools []n.ApplicationGatewayBackendAddressPool) poolsByName {
	indexed := make(poolsByName)
	for _, pool := range pools {
		indexed[backendPoolName(*pool.Name)] = pool
	}
	return indexed
}

func logTarget(verbosity glog.Level, target Target, message string) {
	t, _ := json.Marshal(target)
	glog.V(verbosity).Infof("Target is "+message+": %s", t)
}

// getPoolToTargetsMap creates a map from backend pool to targets this backend pool is responsible for.
// We rely on the configuration that AGIC has already constructed: Frontend Listener, Routing Rules, etc.
// We use the Listener to obtain the target hostname, the RoutingRule to get the
func (c PoolContext) getPoolToTargetsMap() poolToTargets {

	// Index listeners by their name
	listenersByName := make(map[listenerName]*n.ApplicationGatewayHTTPListener)
	for _, listener := range c.Listeners {
		listenersByName[listenerName(*listener.Name)] = listener
	}

	// Index URLPathMaps by the path map name
	pathNameToPath := make(map[pathmapName]n.ApplicationGatewayURLPathMap)
	for _, pm := range c.PathMaps {
		pathNameToPath[pathmapName(*pm.Name)] = pm
	}

	poolToTarget := make(poolToTargets)

	for _, rule := range c.RoutingRules {

		listenerName := listenerName(utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID))

		var hostName string
		if listener, found := listenersByName[listenerName]; !found {
			continue
		} else {
			hostName = *listener.HostName
		}

		target := Target{Hostname: hostName}

		if rule.URLPathMap == nil {
			// SSL Redirects do not have BackendAddressPool
			if rule.BackendAddressPool == nil {
				continue
			}
			poolName := backendPoolName(utils.GetLastChunkOfSlashed(*rule.BackendAddressPool.ID))
			poolToTarget[poolName] = append(poolToTarget[poolName], target)
		} else {
			// Follow the path map
			pathMapName := pathmapName(utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID))
			for _, pathRule := range *pathNameToPath[pathMapName].PathRules {
				if pathRule.BackendAddressPool == nil {
					glog.Errorf("Path Rule %+v does not have BackendAddressPool", *pathRule.Name)
					continue
				}
				poolName := backendPoolName(utils.GetLastChunkOfSlashed(*pathRule.BackendAddressPool.ID))
				if pathRule.Paths == nil {
					glog.V(5).Infof("Path Rule %+v does not have paths list", *pathRule.Name)
					continue
				}
				for _, path := range *pathRule.Paths {
					target.Path = strings.ToLower(path)
					poolToTarget[poolName] = append(poolToTarget[poolName], target)
				}
			}
		}
	}
	return poolToTarget
}
