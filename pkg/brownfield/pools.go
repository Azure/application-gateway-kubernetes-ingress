// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// GetManagedPools filters the given list of backend pools to the list pools that AGIC is allowed to manage.
func GetManagedPools(pools []n.ApplicationGatewayBackendAddressPool, managed []*mtv1.AzureIngressManagedTarget, prohibited []*ptv1.AzureIngressProhibitedTarget, ctx PoolContext) []n.ApplicationGatewayBackendAddressPool {
	blacklist := GetProhibitedTargetList(prohibited)
	whitelist := GetManagedTargetList(managed)

	if len(*blacklist) == 0 && len(*whitelist) == 0 {
		// There is neither TargetBlacklist nor TargetWhitelist -- AGIC will manage all.
		return pools
	}

	// Ignore the TargetWhitelist if TargetBlacklist exists
	if len(*blacklist) > 0 {
		return ctx.applyBlacklist(pools, blacklist)
	}
	return ctx.applyWhitelist(pools, whitelist)
}

// PruneManagedPools removes the managed pools from the given list of pools; resulting in a list of pools not managed by AGIC.
func PruneManagedPools(pools []n.ApplicationGatewayBackendAddressPool, managedTargets []*mtv1.AzureIngressManagedTarget, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget, ctx PoolContext) []n.ApplicationGatewayBackendAddressPool {
	managedPools := GetManagedPools(pools, managedTargets, prohibitedTargets, ctx)
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
	indexed := make(map[backendPoolName]n.ApplicationGatewayBackendAddressPool)
	for _, pool := range pools {
		indexed[backendPoolName(*pool.Name)] = pool
	}
	return indexed
}

func (c PoolContext) applyBlacklist(pools []n.ApplicationGatewayBackendAddressPool, blacklist TargetBlacklist) []n.ApplicationGatewayBackendAddressPool {
	poolToTarget := c.getPoolToTargets()
	managedPools := make(poolsByName)

	for _, pool := range pools {
		for _, target := range poolToTarget[backendPoolName(*pool.Name)] {
			if target.IsIn(blacklist) {
				logTarget(5, target, "in blacklist")
				continue
			}
			logTarget(5, target, "implicitly managed")
			managedPools[backendPoolName(*pool.Name)] = pool
		}
	}
	return poolsMapToList(managedPools)
}

func (c PoolContext) applyWhitelist(pools []n.ApplicationGatewayBackendAddressPool, whitelist TargetWhitelist) []n.ApplicationGatewayBackendAddressPool {
	poolToTarget := c.getPoolToTargets()
	managedPools := make(poolsByName)

	for _, pool := range pools {
		for _, target := range poolToTarget[backendPoolName(*pool.Name)] {
			if target.IsIn(whitelist) {
				logTarget(5, target, "in whitelist")
				managedPools[backendPoolName(*pool.Name)] = pool
				continue
			}
			logTarget(5, target, "NOT in whitelist")
		}
	}
	return poolsMapToList(managedPools)
}

func logTarget(verbosity glog.Level, target Target, message string) {
	t, _ := target.MarshalJSON()
	glog.V(verbosity).Infof("Target is "+message+": %s", t)
}

// getPoolToTargets creates a map from backend pool to targets this backend pool is responsible for.
// We rely on the configuration that AGIC has already constructed: Frontend Listener, Routing Rules, etc.
// We use the Listener to obtain the target hostname, the RoutingRule to get the
func (c PoolContext) getPoolToTargets() poolToTargets {

	listenerMap := c.listenersByName()
	pathNameToPath := c.pathsByName()

	poolToTarget := make(poolToTargets)

	for _, rule := range c.RoutingRules {

		listenerName := listenerName(utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID))

		var hostName string
		if listener, found := listenerMap[listenerName]; !found {
			continue
		} else {
			hostName = *listener.HostName
		}

		target := Target{
			Hostname: hostName,
			Port:     portFromListener(listenerMap[listenerName]),
		}

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
					target.Path = to.StringPtr(NormalizePath(path))
					poolToTarget[poolName] = append(poolToTarget[poolName], target)
				}
			}
		}
	}
	return poolToTarget
}

func poolsMapToList(pools poolsByName) []n.ApplicationGatewayBackendAddressPool {
	var managedPools []n.ApplicationGatewayBackendAddressPool
	for _, pool := range pools {
		managedPools = append(managedPools, pool)
	}
	return managedPools
}

// listenersByName indexes HTTPListeners by their name.
func (c PoolContext) listenersByName() map[listenerName]*n.ApplicationGatewayHTTPListener {
	// Index listeners by their Name
	listenerMap := make(map[listenerName]*n.ApplicationGatewayHTTPListener)
	for _, listener := range c.Listeners {
		listenerMap[listenerName(*listener.Name)] = listener
	}
	return listenerMap
}

// pathsByName indexes URLPathMaps by their name.
func (c PoolContext) pathsByName() map[pathmapName]n.ApplicationGatewayURLPathMap {
	pathNameToPath := make(map[pathmapName]n.ApplicationGatewayURLPathMap)
	for _, pm := range c.PathMaps {
		pathNameToPath[pathmapName(*pm.Name)] = pm
	}
	return pathNameToPath
}
