package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

type PoolSet map[string]interface{}

// PruneHealthProbes filters out probes that are for prohibited targets.
func PruneHealthProbes(probes *[]n.ApplicationGatewayProbe, poolSet PoolSet) {
	// TODO(draychev): flesh out
	var filteredProbes []n.ApplicationGatewayProbe
	for _, probe := range *probes {
		filteredProbes = append(filteredProbes, probe)
	}
	*probes = filteredProbes
}

// PruneRoutingRules filters out rules that point to a BackendAddressPool that does not exist.
func PruneRoutingRules(requestRoutingRules *[]n.ApplicationGatewayRequestRoutingRule, poolSet PoolSet) {
	var filteredRules []n.ApplicationGatewayRequestRoutingRule
	for _, rule := range *requestRoutingRules {
		if rule.BackendAddressPool != nil {
			poolName := utils.GetLastChunkOfSlashed(*rule.BackendAddressPool.ID)
			if _, exists := poolSet[poolName]; !exists {
				continue
			}
		}
		filteredRules = append(filteredRules, rule)
	}
	*requestRoutingRules = filteredRules
}

// PrunePathMaps filters out path maps for non existent backend pools.
func PrunePathMaps(pms *[]n.ApplicationGatewayURLPathMap, poolSet PoolSet) {
	var pathMaps []n.ApplicationGatewayURLPathMap
	for _, pm := range *pms {
		if pm.PathRules == nil {
			pathMaps = append(pathMaps, pm)
			continue
		}
		filterPathRules(&pm, poolSet)
		pathMaps = append(pathMaps, pm)
	}
	*pms = pathMaps
}

func filterPathRules(pm *n.ApplicationGatewayURLPathMap, poolSet map[string]interface{}) {
	var pathRules []n.ApplicationGatewayPathRule

	for _, pr := range *pm.PathRules {
		if pr.BackendAddressPool == nil {
			// keep it
			pathRules = append(pathRules, pr)
			continue
		}
		poolName := utils.GetLastChunkOfSlashed(*pr.BackendAddressPool.ID)
		if _, exists := poolSet[poolName]; !exists {
			// drop it
			continue
		}
		pathRules = append(pathRules, pr)
	}

	*pm.PathRules = pathRules
}
