package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// GetPoolToTargetMapping creates a map from backend pool to target objects.
func GetPoolToTargetMapping(listeners []*n.ApplicationGatewayHTTPListener, requestRoutingRules []n.ApplicationGatewayRequestRoutingRule, paths []n.ApplicationGatewayURLPathMap) map[string]Target {
	listenerMap := make(map[string]*n.ApplicationGatewayHTTPListener)
	for _, listener := range listeners {
		listenerMap[*listener.Name] = listener
	}

	poolToTarget := make(map[string]Target)

	pathMap := make(map[string]n.ApplicationGatewayURLPathMap)
	for _, path := range paths {
		pathMap[*path.Name] = path
	}

	for _, rule := range requestRoutingRules {
		listenerName := utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID)
		if rule.URLPathMap == nil {
			// SSL Redirects won't have BackendAddressPool
			if rule.BackendAddressPool != nil {
				poolName := utils.GetLastChunkOfSlashed(*rule.BackendAddressPool.ID)
				poolToTarget[poolName] = Target{
					Hostname: *listenerMap[listenerName].HostName,
					Port:     portFromListener(listenerMap[listenerName]),
				}
			}
		} else {
			// Follow the path map
			pathMapName := utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID)
			for _, pathRule := range *pathMap[pathMapName].PathRules {
				for _, path := range *pathRule.Paths {
					poolName := utils.GetLastChunkOfSlashed(*pathRule.BackendAddressPool.ID)
					poolToTarget[poolName] = Target{
						Hostname: *listenerMap[listenerName].HostName,
						Port:     portFromListener(listenerMap[listenerName]),
						Path:     &path,
					}
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
