// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

type redirectName string
type redirectsByName map[redirectName]n.ApplicationGatewayRedirectConfiguration

// GetBlacklistedRedirects removes the managed redirects from the given list of redirects; resulting in a list of redirects not managed by AGIC.
func (er ExistingResources) GetBlacklistedRedirects() ([]n.ApplicationGatewayRedirectConfiguration, []n.ApplicationGatewayRedirectConfiguration) {
	blacklisted := er.getBlacklistedRedirectsSet()
	var blacklistedRedirects []n.ApplicationGatewayRedirectConfiguration
	var nonBlacklistedRedirects []n.ApplicationGatewayRedirectConfiguration
	for _, redirect := range er.Redirects {
		if _, isBlacklisted := blacklisted[redirectName(*redirect.Name)]; isBlacklisted {
			blacklistedRedirects = append(blacklistedRedirects, redirect)
			glog.V(5).Infof("[brownfield] Redirect %s is blacklisted", *redirect.Name)
			continue
		}
		glog.V(5).Infof("[brownfield] Redirect %s is not blacklisted", *redirect.Name)
		nonBlacklistedRedirects = append(nonBlacklistedRedirects, redirect)
	}
	return blacklistedRedirects, nonBlacklistedRedirects
}

// LogRedirects emits a few log lines detailing what Redirects are created, blacklisted, and removed from ARM.
func LogRedirects(existingBlacklisted []n.ApplicationGatewayRedirectConfiguration, existingNonBlacklisted []n.ApplicationGatewayRedirectConfiguration, managedRedirects []n.ApplicationGatewayRedirectConfiguration) {
	var garbage []n.ApplicationGatewayRedirectConfiguration

	blacklistedSet := indexRedirectsByName(existingBlacklisted)
	managedSet := indexRedirectsByName(managedRedirects)

	for redirectName, redirect := range indexRedirectsByName(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[redirectName]
		_, existsInNewRedirects := managedSet[redirectName]
		if !existsInBlacklist && !existsInNewRedirects {
			garbage = append(garbage, redirect)
		}
	}

	glog.V(3).Info("[brownfield] Redirects AGIC created: ", getRedirectNames(managedRedirects))
	glog.V(3).Info("[brownfield] Existing Blacklisted Redirects AGIC will retain: ", getRedirectNames(existingBlacklisted))
	glog.V(3).Info("[brownfield] Existing Redirects AGIC will remove: ", getRedirectNames(garbage))
}

// MergeRedirects merges list of lists of redirects into a single list, maintaining uniqueness.
func MergeRedirects(redirectBuckets ...[]n.ApplicationGatewayRedirectConfiguration) []n.ApplicationGatewayRedirectConfiguration {
	uniqRedirects := make(redirectsByName)
	for _, bucket := range redirectBuckets {
		for _, redirect := range bucket {
			uniqRedirects[redirectName(*redirect.Name)] = redirect
		}
	}
	var merged []n.ApplicationGatewayRedirectConfiguration
	for _, redirect := range uniqRedirects {
		merged = append(merged, redirect)
	}
	return merged
}

func getRedirectNames(redirects []n.ApplicationGatewayRedirectConfiguration) string {
	var names []string
	for _, redirect := range redirects {
		names = append(names, *redirect.Name)
	}
	if len(names) == 0 {
		return "n/a"
	}
	return strings.Join(names, ", ")
}

func indexRedirectsByName(redirects []n.ApplicationGatewayRedirectConfiguration) redirectsByName {
	indexed := make(redirectsByName)
	for _, redirect := range redirects {
		indexed[redirectName(*redirect.Name)] = redirect
	}
	return indexed
}

func (er ExistingResources) getBlacklistedRedirectsSet() map[redirectName]interface{} {
	blacklistedRoutingRules, _ := er.GetBlacklistedRoutingRules()
	blacklisted := make(map[redirectName]interface{})
	for _, rule := range blacklistedRoutingRules {
		if rule.RedirectConfiguration != nil && rule.RedirectConfiguration.ID != nil {
			redirectName := redirectName(utils.GetLastChunkOfSlashed(*rule.RedirectConfiguration.ID))
			blacklisted[redirectName] = nil
		}
	}

	blacklistedPathMaps, _ := er.GetBlacklistedPathMaps()
	for _, pathMap := range blacklistedPathMaps {
		if pathMap.DefaultRedirectConfiguration != nil && pathMap.DefaultRedirectConfiguration.ID != nil {
			redirectName := redirectName(utils.GetLastChunkOfSlashed(*pathMap.DefaultRedirectConfiguration.ID))
			blacklisted[redirectName] = nil
		}
		if pathMap.PathRules == nil {
			glog.Errorf("PathMap %s does not have PathRules", *pathMap.Name)
			continue
		}
		for _, rule := range *pathMap.PathRules {
			if rule.RedirectConfiguration != nil && rule.RedirectConfiguration.ID != nil {
				redirectName := redirectName(utils.GetLastChunkOfSlashed(*rule.RedirectConfiguration.ID))
				blacklisted[redirectName] = nil
			}
		}
	}

	return blacklisted
}
