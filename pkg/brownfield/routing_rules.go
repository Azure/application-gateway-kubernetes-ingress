// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"errors"
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

func (er ExistingResources) getHostNameForRoutingRule(rule n.ApplicationGatewayRequestRoutingRule) (string, error) {
	listenerName := listenerName(utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID))
	if listener, found := er.getListenersByName()[listenerName]; !found {
		glog.Errorf("[brownfield] Could not find listener %s in index", listenerName)
		// TODO(draychev): move this error into a top-level file
		return "", errors.New("failed looking up listener")
	} else if listener.HostName != nil {
		return *listener.HostName, nil
	}
	return "", nil
}

// getRuleToTargets creates a map from backend pool to targets this backend pool is responsible for.
// We rely on the configuration that AGIC has already constructed: Frontend Listener, Routing Rules, etc.
// We use the Listener to obtain the target hostname, the RoutingRule to get the URL etc.
func (er ExistingResources) getRuleToTargets() (ruleToTargets, pathmapToTargets) {
	ruleToTargets := make(ruleToTargets)
	pathMapToTargets := make(pathmapToTargets)
	for _, rule := range er.RoutingRules {
		if rule.HTTPListener == nil || rule.HTTPListener.ID == nil {
			continue
		}
		hostName, err := er.getHostNameForRoutingRule(rule)
		if err != nil {
			glog.Errorf("[brownfield] Could not obtain hostname for rule %s; Skipping rule", ruleName(*rule.Name))
			continue
		}

		// Regardless of whether we have a URL PathMap or not. This matches the default backend pool.
		ruleToTargets[ruleName(*rule.Name)] = append(ruleToTargets[ruleName(*rule.Name)], Target{
			Hostname: hostName,
			// Path deliberately omitted
		})

		// SSL Redirects do not have BackendAddressPool
		if rule.URLPathMap != nil {
			// Follow the path map
			pathMapName, pathRules := er.getPathRules(rule)
			for _, pathRule := range pathRules {
				if pathRule.Paths == nil {
					glog.V(5).Infof("[brownfield] Path Rule %+v does not have paths list", *pathRule.Name)
					continue
				}
				for _, path := range *pathRule.Paths {
					target := Target{hostName, strings.ToLower(path)}
					ruleToTargets[ruleName(*rule.Name)] = append(ruleToTargets[ruleName(*rule.Name)], target)
					pathMapToTargets[pathMapName] = append(pathMapToTargets[pathMapName], target)
				}
			}
		}
	}
	return ruleToTargets, pathMapToTargets
}

func (er ExistingResources) getPathRules(rule n.ApplicationGatewayRequestRoutingRule) (urlPathMapName, []n.ApplicationGatewayPathRule) {
	pathMapName := urlPathMapName(utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID))
	pathMapsByName := er.getURLPathMapsByName()
	if pathMap, ok := pathMapsByName[pathMapName]; ok {
		return pathMapName, *pathMap.PathRules
	}
	glog.Errorf("[brownfield] Did not find URLPathMap with ID %s", pathMapName)
	return pathMapName, []n.ApplicationGatewayPathRule{}
}
