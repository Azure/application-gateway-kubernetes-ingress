// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

type backendPoolName string
type poolsByName map[backendPoolName]n.ApplicationGatewayBackendAddressPool

// GetBlacklistedPools removes the managed pools from the given list of pools; resulting in a list of pools not managed by AGIC.
func (er ExistingResources) GetBlacklistedPools() ([]n.ApplicationGatewayBackendAddressPool, []n.ApplicationGatewayBackendAddressPool) {
	blacklistedPoolsSet := er.getBlacklistedPoolsSet()
	var blacklistedPools []n.ApplicationGatewayBackendAddressPool
	var nonBlacklistedPools []n.ApplicationGatewayBackendAddressPool
	for _, pool := range er.BackendPools {
		if _, isBlacklisted := blacklistedPoolsSet[backendPoolName(*pool.Name)]; isBlacklisted {
			blacklistedPools = append(blacklistedPools, pool)
			klog.V(5).Infof("[brownfield] Backend Address Pool %s is blacklisted", *pool.Name)
			continue
		}
		klog.V(5).Infof("[brownfield] Backend Address Pool %s is NOT blacklisted", *pool.Name)
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

	klog.V(3).Info("[brownfield] Pools AGIC created: ", getPoolNames(managedPools))
	klog.V(3).Info("[brownfield] Existing Blacklisted Pools AGIC will retain: ", getPoolNames(existingBlacklisted))
	klog.V(3).Info("[brownfield] Existing Pools AGIC will remove: ", getPoolNames(garbage))
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

func (er ExistingResources) getBlacklistedPoolsSet() map[backendPoolName]interface{} {
	blacklistedRoutingRules, _ := er.GetBlacklistedRoutingRules()
	blacklistedPoolsSet := make(map[backendPoolName]interface{})
	for _, rule := range blacklistedRoutingRules {
		if rule.BackendAddressPool != nil && rule.BackendAddressPool.ID != nil {
			poolName := backendPoolName(utils.GetLastChunkOfSlashed(*rule.BackendAddressPool.ID))
			blacklistedPoolsSet[poolName] = nil
		}
	}

	blacklistedPathMaps, _ := er.GetBlacklistedPathMaps()
	for _, pathMap := range blacklistedPathMaps {
		if pathMap.DefaultBackendAddressPool != nil && pathMap.DefaultBackendAddressPool.ID != nil {
			poolName := backendPoolName(utils.GetLastChunkOfSlashed(*pathMap.DefaultBackendAddressPool.ID))
			blacklistedPoolsSet[poolName] = nil
		}
		if pathMap.PathRules == nil {
			klog.Errorf("PathMap %s does not have PathRules", *pathMap.Name)
			continue
		}
		for _, rule := range *pathMap.PathRules {
			if rule.BackendAddressPool != nil && rule.BackendAddressPool.ID != nil {
				poolName := backendPoolName(utils.GetLastChunkOfSlashed(*rule.BackendAddressPool.ID))
				blacklistedPoolsSet[poolName] = nil
			}
		}
	}

	return blacklistedPoolsSet
}
