// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

type redirectName string
type redirectsByName map[redirectName]n.ApplicationGatewayRedirectConfiguration

// GetBlacklistedRedirects filters the given list of health probes to the list Probes that AGIC is allowed to manage.
func (er ExistingResources) GetBlacklistedRedirects() ([]n.ApplicationGatewayRedirectConfiguration, []n.ApplicationGatewayRedirectConfiguration) {
	blacklistedListeners := er.getBlacklistedListenersSet()
	var blacklisted, nonBlacklisted []n.ApplicationGatewayRedirectConfiguration
	for _, redirect := range er.Redirects {
		// We consider a redirect blacklisted if it is pointing (targeting) a listener that is blacklisted
		listenerNm := listenerName(utils.GetLastChunkOfSlashed(*redirect.TargetListener.ID))
		if _, exists := blacklistedListeners[listenerNm]; exists {
			glog.V(5).Infof("[brownfield] Redirect %s is blacklisted", *redirect.Name)
			blacklisted = append(blacklisted, redirect)
			continue
		}
		glog.V(5).Infof("[brownfield] Redirect %s is not blacklisted", *redirect.Name)
		nonBlacklisted = append(nonBlacklisted, redirect)
	}
	return blacklisted, nonBlacklisted
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
