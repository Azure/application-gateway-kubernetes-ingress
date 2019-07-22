// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"
)

type urlPathMapName string
type pathRuleName string
type pathmapToTargets map[urlPathMapName][]Target
type pathRulesByName map[pathRuleName]n.ApplicationGatewayPathRule
type pathMapsByName map[urlPathMapName]n.ApplicationGatewayURLPathMap

// GetBlacklistedPathMaps filters the given list of routing pathMaps to the list pathMaps that AGIC is allowed to manage.
func (er ExistingResources) GetBlacklistedPathMaps() ([]n.ApplicationGatewayURLPathMap, []n.ApplicationGatewayURLPathMap) {

	blacklist := GetTargetBlacklist(er.ProhibitedTargets)
	if blacklist == nil {
		return nil, er.URLPathMaps
	}
	_, pathMapToTargets := er.getRuleToTargets()
	glog.V(5).Infof("[brownfield] PathMap to Targets map: %+v", pathMapToTargets)

	// Figure out if the given BackendAddressPathMap is blacklisted. It will be if it has a host/path that
	// has been referenced in a AzureIngressProhibitedTarget CRD (even if it has some other paths that are not)
	isBlacklisted := func(pathMap n.ApplicationGatewayURLPathMap) bool {
		targetsForPathMap := pathMapToTargets[urlPathMapName(*pathMap.Name)]
		for _, target := range targetsForPathMap {
			if target.IsBlacklisted(blacklist) {
				glog.V(5).Infof("[brownfield] Routing PathMap %s is blacklisted", *pathMap.Name)
				return true
			}
		}
		glog.V(5).Infof("[brownfield] Routing PathMap %s is NOT blacklisted", *pathMap.Name)
		return false
	}

	var blacklistedPathMaps []n.ApplicationGatewayURLPathMap
	var nonBlacklistedPathMaps []n.ApplicationGatewayURLPathMap
	for _, pathMap := range er.URLPathMaps {
		if isBlacklisted(pathMap) {
			blacklistedPathMaps = append(blacklistedPathMaps, pathMap)
			continue
		}
		nonBlacklistedPathMaps = append(nonBlacklistedPathMaps, pathMap)
	}
	return blacklistedPathMaps, nonBlacklistedPathMaps
}

// MergePathMaps merges list of lists of pathMaps into a single list, maintaining uniqueness.
func MergePathMaps(pathMapBuckets ...[]n.ApplicationGatewayURLPathMap) []n.ApplicationGatewayURLPathMap {
	uniq := make(pathMapsByName)
	for _, bucket := range pathMapBuckets {
		for _, pathMap := range bucket {
			if existingPathMap, exists := uniq[urlPathMapName(*pathMap.Name)]; exists {
				glog.Infof("[brownfield] Merging urlpath %s in existing blacklist and AGIC list", *existingPathMap.Name)
				uniq[urlPathMapName(*pathMap.Name)].PathRules = MergePathRules(existingPathMap.PathRules, pathMap.PathRules)
			} else {
				glog.Infof("[brownfield] Adding urlpath %s to the uniq map", *pathMap.Name)
				uniq[urlPathMapName(*pathMap.Name)] = pathMap
			}
		}
	}
	var merged []n.ApplicationGatewayURLPathMap
	for _, pathMap := range uniq {
		merged = append(merged, pathMap)
	}
	return merged
}

// MergePathMapsWithBasicRule merges a Url pathmap with a basic routing rule
func MergePathMapsWithBasicRule(pathMap *n.ApplicationGatewayURLPathMap, rule *n.ApplicationGatewayRequestRoutingRule) *n.ApplicationGatewayURLPathMap {
	// Replace the default values in the path map with values from the routing rule
	pathMap.DefaultBackendAddressPool = rule.BackendAddressPool
	pathMap.DefaultBackendHTTPSettings = rule.BackendHTTPSettings
	pathMap.DefaultRedirectConfiguration = rule.RedirectConfiguration
	pathMap.DefaultRewriteRuleSet = rule.RewriteRuleSet
	return pathMap
}

// MergePathRules merges list of lists of pathMaps into a single list, maintaining uniqueness.
func MergePathRules(pathRulesBucket ...*[]n.ApplicationGatewayPathRule) *[]n.ApplicationGatewayPathRule {
	uniq := make(pathRulesByName)
	for _, bucket := range pathRulesBucket {
		for _, pathRule := range *bucket {
			uniq[pathRuleName(*pathRule.Name)] = pathRule
		}
	}
	var merged []n.ApplicationGatewayPathRule
	for _, pathRule := range uniq {
		merged = append(merged, pathRule)
	}
	return &merged
}

// LookupPathMap using resourceID in the list of path map
func LookupPathMap(pathMaps *[]n.ApplicationGatewayURLPathMap, resourceID *string) *n.ApplicationGatewayURLPathMap {
	for idx, pathMap := range *pathMaps {
		if *pathMap.ID == *resourceID {
			return &(*pathMaps)[idx]
		}
	}

	return nil
}

// DeletePathMap deletes a path map from the list of path maps and return the new list
func DeletePathMap(pathMapsPtr *[]n.ApplicationGatewayURLPathMap, resourceID *string) *[]n.ApplicationGatewayURLPathMap {
	pathMaps := *pathMapsPtr
	deleteIdx := -1
	for idx, pathMap := range pathMaps {
		if *pathMap.ID == *resourceID {
			glog.V(5).Infof("[brownfield] Deleting %s", *resourceID)
			deleteIdx = idx
		}
	}

	if deleteIdx != -1 {
		pathMaps = append(pathMaps[:deleteIdx], pathMaps[deleteIdx+1:]...)
	}

	return &pathMaps
}

// LogPathMaps emits a few log lines detailing what pathMaps are created, blacklisted, and removed from ARM.
func LogPathMaps(existingBlacklisted []n.ApplicationGatewayURLPathMap, existingNonBlacklisted []n.ApplicationGatewayURLPathMap, managedPathMaps []n.ApplicationGatewayURLPathMap) {
	var garbage []n.ApplicationGatewayURLPathMap

	blacklistedSet := indexPathMapsByName(existingBlacklisted)
	managedSet := indexPathMapsByName(managedPathMaps)

	for pathMapName, pathMap := range indexPathMapsByName(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[pathMapName]
		_, existsInNewPathMaps := managedSet[pathMapName]
		if !existsInBlacklist && !existsInNewPathMaps {
			garbage = append(garbage, pathMap)
		}
	}

	glog.V(3).Info("[brownfield] PathMaps AGIC created: ", getPathMapNames(managedPathMaps))
	glog.V(3).Info("[brownfield] Existing Blacklisted PathMaps AGIC will retain: ", getPathMapNames(existingBlacklisted))
	glog.V(3).Info("[brownfield] Existing PathMaps AGIC will remove: ", getPathMapNames(garbage))
}

func getPathMapNames(pathMaps []n.ApplicationGatewayURLPathMap) string {
	var names []string
	for _, pathMap := range pathMaps {
		names = append(names, *pathMap.Name)
	}
	if len(names) == 0 {
		return "n/a"
	}
	return strings.Join(names, ", ")
}

func indexPathMapsByName(pathMaps []n.ApplicationGatewayURLPathMap) pathMapsByName {
	indexed := make(pathMapsByName)
	for _, pathMap := range pathMaps {
		indexed[urlPathMapName(*pathMap.Name)] = pathMap
	}
	return indexed
}

func (er ExistingResources) getURLPathMapsByName() pathMapsByName {
	if er.urlPathMapsByName != nil {
		return er.urlPathMapsByName
	}
	// Index URLPathMaps by the path map name
	urlpathMapsByName := make(pathMapsByName)
	for _, pm := range er.URLPathMaps {
		urlpathMapsByName[urlPathMapName(*pm.Name)] = pm
	}

	er.urlPathMapsByName = urlpathMapsByName
	return urlpathMapsByName
}
