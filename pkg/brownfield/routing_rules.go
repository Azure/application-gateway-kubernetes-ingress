// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"github.com/Azure/go-autorest/autorest/to"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

type ruleName string
type rulesByName map[ruleName]n.ApplicationGatewayRequestRoutingRule
type ruleToTargets map[ruleName][]Target

// GetBlacklistedRoutingRules filters the given list of routing rules to the list rules that AGIC is allowed to manage.
func (er ExistingResources) GetBlacklistedRoutingRules() ([]n.ApplicationGatewayRequestRoutingRule, []n.ApplicationGatewayRequestRoutingRule) {

	// TODO(draychev): make a method of ExistingResources
	blacklist := GetTargetBlacklist(er.ProhibitedTargets)
	if blacklist == nil {
		return nil, er.RoutingRules
	}
	ruleToTargets, _ := er.getRuleToTargets()
	glog.V(5).Infof("[brownfield] Rule to Targets map: %+v", ruleToTargets)

	// Figure out if the given routing rule is blacklisted. It will be if it has a host/path that
	// has been referenced in a AzureIngressProhibitedTarget CRD (even if it has some other paths that are not)
	isBlacklisted := func(rule n.ApplicationGatewayRequestRoutingRule) bool {
		targetsForRule := ruleToTargets[ruleName(*rule.Name)]
		for _, target := range targetsForRule {
			if target.IsBlacklisted(blacklist) {
				glog.V(5).Infof("[brownfield] Routing Rule %s is blacklisted", *rule.Name)
				return true
			}
		}
		glog.V(5).Infof("[brownfield] Routing Rule %s is NOT blacklisted", *rule.Name)
		return false
	}

	var blacklistedRules []n.ApplicationGatewayRequestRoutingRule
	var nonBlacklistedRules []n.ApplicationGatewayRequestRoutingRule
	for _, rule := range er.RoutingRules {
		if isBlacklisted(rule) {
			blacklistedRules = append(blacklistedRules, rule)
			continue
		}
		nonBlacklistedRules = append(nonBlacklistedRules, rule)
	}
	return blacklistedRules, nonBlacklistedRules
}

// MergeRules merges list of lists of rules into a single list, maintaining uniqueness.
func MergeRules(ruleBuckets ...[]n.ApplicationGatewayRequestRoutingRule) []n.ApplicationGatewayRequestRoutingRule {
	uniq := make(rulesByName)
	for _, bucket := range ruleBuckets {
		for _, rule := range bucket {
			uniq[ruleName(*rule.Name)] = rule
		}
	}
	var merged []n.ApplicationGatewayRequestRoutingRule
	for _, rule := range uniq {
		merged = append(merged, rule)
	}
	return merged
}

// LogRules emits a few log lines detailing what rules are created, blacklisted, and removed from ARM.
func LogRules(existingBlacklisted []n.ApplicationGatewayRequestRoutingRule, existingNonBlacklisted []n.ApplicationGatewayRequestRoutingRule, managedRules []n.ApplicationGatewayRequestRoutingRule) {
	var garbage []n.ApplicationGatewayRequestRoutingRule

	blacklistedSet := indexRulesByName(existingBlacklisted)
	managedSet := indexRulesByName(managedRules)

	for ruleName, rule := range indexRulesByName(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[ruleName]
		_, existsInNewRules := managedSet[ruleName]
		if !existsInBlacklist && !existsInNewRules {
			garbage = append(garbage, rule)
		}
	}

	glog.V(3).Info("[brownfield] Rules AGIC created: ", getRuleNames(managedRules))
	glog.V(3).Info("[brownfield] Existing Blacklisted Rules AGIC will retain: ", getRuleNames(existingBlacklisted))
	glog.V(3).Info("[brownfield] Existing Rules AGIC will remove: ", getRuleNames(garbage))
}

func getRuleNames(cert []n.ApplicationGatewayRequestRoutingRule) string {
	var names []string
	for _, p := range cert {
		names = append(names, *p.Name)
	}
	if len(names) == 0 {
		return "n/a"
	}
	return strings.Join(names, ", ")
}

func indexRulesByName(rules []n.ApplicationGatewayRequestRoutingRule) rulesByName {
	indexed := make(rulesByName)
	for _, rule := range rules {
		indexed[ruleName(*rule.Name)] = rule
	}
	return indexed
}

func (er ExistingResources) getHostNameForRoutingRule(rule n.ApplicationGatewayRequestRoutingRule) *string {
	listenerName := listenerName(utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID))
	if listener, found := er.getListenersByName()[listenerName]; !found {
		return nil
	} else if listener.HostName != nil {
		return listener.HostName
	}
	return to.StringPtr("")
}

// getRuleToTargets creates a map from backend pool to targets this backend pool is responsible for.
// We rely on the configuration that AGIC has already constructed: Frontend Listener, Routing Rules, etc.
// We use the Listener to obtain the target hostname, the RoutingRule to get the URL etc.
func (er ExistingResources) getRuleToTargets() (ruleToTargets, pathmapToTargets) {
	ruleToTargets := make(ruleToTargets)
	pathMapToTargets := make(pathmapToTargets)

	// Index URLPathMaps by the path map name
	pathNameToPath := make(map[urlPathMapName]n.ApplicationGatewayURLPathMap)
	for _, pm := range er.URLPathMaps {
		pathNameToPath[urlPathMapName(*pm.Name)] = pm
	}

	for _, rule := range er.RoutingRules {
		if rule.HTTPListener == nil || rule.HTTPListener.ID == nil {
			continue
		}
		ruleNm := ruleName(*rule.Name)
		hostName := er.getHostNameForRoutingRule(rule)
		if hostName == nil {
			continue
		}

		// Regardless of whether we have a URL PathMap or not. This matches the default backend pool.
		target := Target{
			Hostname: *hostName,
			// Path deliberately omitted
		}
		ruleToTargets[ruleNm] = append(ruleToTargets[ruleNm], target)

		if rule.URLPathMap == nil {
			// SSL Redirects do not have BackendAddressPool
			continue
		}
		// Follow the path map
		pathMapName := urlPathMapName(utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID))

		for _, pathRule := range *pathNameToPath[pathMapName].PathRules {
			if pathRule.Paths == nil {
				glog.V(5).Infof("[brownfield] Path Rule %+v does not have paths list", *pathRule.Name)
				continue
			}

			for _, path := range *pathRule.Paths {
				target := Target{
					Hostname: *hostName,
					Path:     strings.ToLower(path),
				}
				ruleToTargets[ruleNm] = append(ruleToTargets[ruleNm], target)
				pathMapToTargets[pathMapName] = append(pathMapToTargets[pathMapName], target)
			}
		}
	}
	return ruleToTargets, pathMapToTargets
}
