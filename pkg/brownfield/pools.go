package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// GetPoolToTargetMapping creates a map from backend pool to target objects.
func GetPoolToTargetMapping(listeners []*n.ApplicationGatewayHTTPListener, requestRoutingRules []n.ApplicationGatewayRequestRoutingRule, pathMaps []n.ApplicationGatewayURLPathMap) map[string]Target {
	listenerMap := make(map[string]*n.ApplicationGatewayHTTPListener)
	for _, listener := range listeners {
		listenerMap[*listener.Name] = listener
	}

	poolToTarget := make(map[string]Target)

	pathNameToPath := make(map[string]n.ApplicationGatewayURLPathMap)
	for _, pm := range pathMaps {
		pathNameToPath[*pm.Name] = pm
	}

	for _, rule := range requestRoutingRules {
		listenerName := utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID)
		var hostName string
		if listener, found := listenerMap[listenerName]; !found {
			continue
		} else {
			hostName = *listener.HostName
		}
		port := portFromListener(listenerMap[listenerName])
		target := Target{
			Hostname: hostName,
			Port:     port,
		}

		if rule.URLPathMap == nil {
			// SSL Redirects do not have BackendAddressPool
			if rule.BackendAddressPool == nil {
				continue
			}
			poolName := utils.GetLastChunkOfSlashed(*rule.BackendAddressPool.ID)
			poolToTarget[poolName] = target
		} else {
			// Follow the path map
			pathMapName := utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID)
			for _, pathRule := range *pathNameToPath[pathMapName].PathRules {
				if pathRule.Paths == nil {
					glog.V(5).Infof("Path Rule %+v does not have paths list", *pathRule.Name)
					continue
				}
				for _, path := range *pathRule.Paths {
					poolName := utils.GetLastChunkOfSlashed(*pathRule.BackendAddressPool.ID)
					target.Path = &path
					poolToTarget[poolName] = target
				}
			}
		}
	}
	return poolToTarget
}

func portFromListener(listener *n.ApplicationGatewayHTTPListener) int32 {
	if listener != nil && listener.Protocol == n.HTTPS {
		return int32(443)
	}
	return int32(80)
}

// MergePools merges list of lists of backend address pools into a single list, maintaining uniqueness.
func MergePools(pools ...[]n.ApplicationGatewayBackendAddressPool) []n.ApplicationGatewayBackendAddressPool {
	uniqPool := make(map[string]n.ApplicationGatewayBackendAddressPool)
	for _, bucket := range pools {
		for _, p := range bucket {
			uniqPool[*p.Name] = p
		}
	}
	var merged []n.ApplicationGatewayBackendAddressPool
	for _, pool := range uniqPool {
		merged = append(merged, pool)
	}
	return merged
}

// GetManagedPools returns the list of backend pools that will be managed by AGIC.
func GetManagedPools(pools []n.ApplicationGatewayBackendAddressPool, managedTargets []*mtv1.AzureIngressManagedTarget, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget, listeners []*n.ApplicationGatewayHTTPListener, routingRules []n.ApplicationGatewayRequestRoutingRule, paths []n.ApplicationGatewayURLPathMap) []n.ApplicationGatewayBackendAddressPool {
	blacklist := GetProhibitedTargetList(prohibitedTargets)
	whitelist := GetManagedTargetList(managedTargets)

	if len(*blacklist) == 0 && len(*whitelist) == 0 {
		return pools
	}

	var managedPools []n.ApplicationGatewayBackendAddressPool

	poolToTarget := GetPoolToTargetMapping(listeners, routingRules, paths)

	// Process Blacklist first
	if len(*blacklist) > 0 {
		// Apply blacklist
		for _, pool := range pools {
			target := poolToTarget[*pool.Name]
			if target.IsIn(blacklist) {
				continue
			}
			managedPools = append(managedPools, pool)
		}
		return managedPools
	}

	// Is it whitelisted?
	for _, pool := range pools {
		if poolToTarget[*pool.Name].IsIn(whitelist) {
			managedPools = append(managedPools, pool)
		}
	}

	for _, pool := range pools {
		if poolToTarget[*pool.Name].IsIn(blacklist) {
			managedPools = append(managedPools, pool)
		}
	}
	return managedPools
}

// PruneManagedPools removes the managed pools from the given list and returns a list of pools that is NOT managed by AGIC.
func PruneManagedPools(pools []n.ApplicationGatewayBackendAddressPool, managedTargets []*mtv1.AzureIngressManagedTarget, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget, listeners []*n.ApplicationGatewayHTTPListener, routingRules []n.ApplicationGatewayRequestRoutingRule, paths []n.ApplicationGatewayURLPathMap) []n.ApplicationGatewayBackendAddressPool {
	managedPool := GetManagedPools(pools, managedTargets, prohibitedTargets, listeners, routingRules, paths)
	if managedPool == nil {
		return pools
	}
	indexed := make(map[string]n.ApplicationGatewayBackendAddressPool)
	for _, pool := range managedPool {
		indexed[*pool.Name] = pool
	}
	var unmanagedPools []n.ApplicationGatewayBackendAddressPool
	for _, probe := range pools {
		if _, isManaged := indexed[*probe.Name]; !isManaged {
			unmanagedPools = append(unmanagedPools, probe)
		}
	}
	return unmanagedPools
}
