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

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// GetBlacklistedPools removes the managed pools from the given list of pools; resulting in a list of pools not managed by AGIC.
func (ctx PoolContext) GetBlacklistedPools() ([]n.ApplicationGatewayBackendAddressPool, []n.ApplicationGatewayBackendAddressPool) {
	blacklist := GetTargetBlacklist(ctx.ProhibitedTargets)
	if blacklist == nil {
		return []n.ApplicationGatewayBackendAddressPool{}, ctx.BackendPools
	}
	poolToTargets := ctx.getPoolToTargetsMap()
	glog.V(5).Infof("Backend Pool to Targets map: %+v", poolToTargets)

	// Figure out if the given BackendAddressPool is blacklisted. It will be if it has a host/path that
	// has been referenced in a AzureIngressProhibitedTarget CRD (even if it has some other paths that are not)
	isPoolBlacklisted := func(pool n.ApplicationGatewayBackendAddressPool) bool {
		targetsForPool := poolToTargets[backendPoolName(*pool.Name)]
		for _, target := range targetsForPool {
			if target.IsBlacklisted(blacklist) {
				logTarget(5, target, "in blacklist")
				return true
			}
			logTarget(5, target, "not in blacklist")
		}
		return false
	}

	var blacklistedPools []n.ApplicationGatewayBackendAddressPool
	var nonBlacklistedPools []n.ApplicationGatewayBackendAddressPool
	for _, pool := range ctx.BackendPools {
		if isPoolBlacklisted(pool) {
			blacklistedPools = append(blacklistedPools, pool)
			glog.V(5).Infof("Backend Address Pool %s is blacklisted", *pool.Name)
			continue
		}
		glog.V(5).Infof("Backend Address Pool %s is NOT blacklisted", *pool.Name)
		nonBlacklistedPools = append(nonBlacklistedPools, pool)
	}
	return blacklistedPools, nonBlacklistedPools
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

// LogPools emits a few log lines detailing what pools are created, blacklisted, and removed from ARM.
func LogPools(existingBlacklisted []n.ApplicationGatewayBackendAddressPool, existingNonBlacklisted []n.ApplicationGatewayBackendAddressPool, managedPools []n.ApplicationGatewayBackendAddressPool) {
	var garbage []n.ApplicationGatewayBackendAddressPool

	blacklistedSet := indexPoolsByName(existingBlacklisted)
	managedSet := indexPoolsByName(managedPools)

	for poolName, pool := range indexPoolsByName(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[poolName]
		_, existsInNewPools := managedSet[poolName]
		if !existsInBlacklist && !existsInNewPools {
			garbage = append(garbage, pool)
		}
	}

	glog.V(3).Info("[brownfield] Pools AGIC created: ", getPoolNames(managedPools))
	glog.V(3).Info("[brownfield] Existing Blacklisted Pools AGIC will retain: ", getPoolNames(existingBlacklisted))
	glog.V(3).Info("[brownfield] Existing Pools AGIC will remove: ", getPoolNames(garbage))
}

func indexPoolsByName(pools []n.ApplicationGatewayBackendAddressPool) poolsByName {
	indexed := make(poolsByName)
	for _, pool := range pools {
		indexed[backendPoolName(*pool.Name)] = pool
	}
	return indexed
}

func getPoolNames(pool []n.ApplicationGatewayBackendAddressPool) string {
	var names []string
	for _, p := range pool {
		names = append(names, *p.Name)
	}
	if len(names) == 0 {
		return "n/a"
	}
	return strings.Join(names, ", ")
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
	listenersByName := make(map[listenerName]n.ApplicationGatewayHTTPListener)
	for _, listener := range c.Listeners {
		listenersByName[listenerName(*listener.Name)] = listener
	}

	// Index URLPathMaps by the path map name
	pathNameToPath := make(map[pathmapName]n.ApplicationGatewayURLPathMap)
	for _, pm := range c.PathMaps {
		pathNameToPath[pathmapName(*pm.Name)] = pm
	}

	// Add the default backend pool - with no target. This will be overwritten if the default backend pool exists with
	// some targets already.
	poolToTarget := poolToTargets{
		backendPoolName(*c.DefaultBackendPool.Name): []Target{},
	}

	for _, rule := range c.RoutingRules {
		listenerName := listenerName(utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID))

		var hostName string
		if listener, found := listenersByName[listenerName]; !found {
			continue
		} else if listener.HostName != nil {
			hostName = *listener.HostName
		} else {
			hostName = ""
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

			// In case there are no PathRules
			if pathNameToPath[pathMapName].PathRules == nil {
				if pathNameToPath[pathMapName].DefaultBackendAddressPool == nil {
					glog.Errorf("Path map with name %s does not have PathRules and does not have DefaultBackendAddressPool", pathMapName)
					continue
				}
				poolName := backendPoolName(*c.DefaultBackendPool.Name)
				poolToTarget[poolName] = append(poolToTarget[poolName], target)
			} else {
				// Go through the path rules
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
	}
	return poolToTarget
}
