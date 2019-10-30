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

type listenersByName map[listenerName]n.ApplicationGatewayHTTPListener

// GetBlacklistedListeners filters the given list of health probes to the list Probes that AGIC is allowed to manage.
func (er ExistingResources) GetBlacklistedListeners() ([]n.ApplicationGatewayHTTPListener, []n.ApplicationGatewayHTTPListener) {
	blacklistedListenersSet := er.getBlacklistedListenersSet()
	var blacklisted, nonBlacklisted []n.ApplicationGatewayHTTPListener
	for _, listener := range er.Listeners {
		listenerNm := listenerName(*listener.Name)
		if _, exists := blacklistedListenersSet[listenerNm]; exists {
			glog.V(5).Infof("[brownfield] Listener %s is blacklisted", listenerNm)
			blacklisted = append(blacklisted, listener)
			continue
		}
		glog.V(5).Infof("[brownfield] Listener %s is not blacklisted", listenerNm)
		nonBlacklisted = append(nonBlacklisted, listener)
	}
	return blacklisted, nonBlacklisted
}

type uniqueListenerConfig struct {
	HostName                string
	Protocol                n.ApplicationGatewayProtocol
	FrontendPortID          string
	FrontendIPConfiguration string
}

// MergeListeners merges list of lists of listeners into a single list, maintaining uniqueness.
func MergeListeners(listenerBuckets ...[]n.ApplicationGatewayHTTPListener) []n.ApplicationGatewayHTTPListener {
	uniq := make(map[uniqueListenerConfig]n.ApplicationGatewayHTTPListener)
	for _, bucket := range listenerBuckets {
		for _, listener := range bucket {
			listenerConfig := uniqueListenerConfig{
				Protocol: listener.Protocol,
			}
			if listener.HostName != nil {
				listenerConfig.HostName = *listener.HostName
			}
			if listener.FrontendIPConfiguration != nil && listener.FrontendIPConfiguration.ID != nil {
				listenerConfig.FrontendIPConfiguration = *listener.FrontendIPConfiguration.ID
			}
			if listener.FrontendPort != nil && listener.FrontendPort.ID != nil {
				listenerConfig.FrontendPortID = *listener.FrontendPort.ID
			}
			if _, exists := uniq[listenerConfig]; !exists {
				uniq[listenerConfig] = listener
			}
		}
	}
	var merged []n.ApplicationGatewayHTTPListener
	for _, listener := range uniq {
		merged = append(merged, listener)
	}
	return merged
}

// LogListeners emits a few log lines detailing what Listeners are created, blacklisted, and removed from ARM.
func LogListeners(existingBlacklisted []n.ApplicationGatewayHTTPListener, existingNonBlacklisted []n.ApplicationGatewayHTTPListener, managedListeners []n.ApplicationGatewayHTTPListener) {
	var garbage []n.ApplicationGatewayHTTPListener

	blacklistedSet := indexListenersByName(existingBlacklisted)
	managedSet := indexListenersByName(managedListeners)

	for listenerName, listener := range indexListenersByName(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[listenerName]
		_, existsInNewListeners := managedSet[listenerName]
		if !existsInBlacklist && !existsInNewListeners {
			garbage = append(garbage, listener)
		}
	}

	glog.V(3).Info("[brownfield] Listeners AGIC created: ", getListenerNames(managedListeners))
	glog.V(3).Info("[brownfield] Existing Blacklisted Listeners AGIC will retain: ", getListenerNames(existingBlacklisted))
	glog.V(3).Info("[brownfield] Existing Listeners AGIC will remove: ", getListenerNames(garbage))
}

func getListenerNames(listeners []n.ApplicationGatewayHTTPListener) string {
	var names []string
	for _, p := range listeners {
		names = append(names, *p.Name)
	}
	if len(names) == 0 {
		return "n/a"
	}
	return strings.Join(names, ", ")
}

func indexListenersByName(listeners []n.ApplicationGatewayHTTPListener) listenersByName {
	indexed := make(listenersByName)
	for _, listener := range listeners {
		indexed[listenerName(*listener.Name)] = listener
	}
	return indexed
}

// getListenersByName indexes listeners by their name
func (er *ExistingResources) getListenersByName() listenersByName {
	if er.listenersByName != nil {
		return er.listenersByName
	}
	listenersByName := make(listenersByName)
	for _, listener := range er.Listeners {
		listenersByName[listenerName(listenerName(*listener.Name))] = listener
	}
	er.listenersByName = listenersByName
	return listenersByName
}

func (er ExistingResources) getBlacklistedListenersSet() map[listenerName]interface{} {
	// Determine the list of prohibited listeners from the hostnames
	blacklistedListenersSet := make(map[listenerName]interface{})
	prohibitedHostnames := er.getProhibitedHostnames()
	for _, listener := range er.Listeners {
		if listener.HostName == nil {
			continue
		}
		if _, exists := prohibitedHostnames[*listener.HostName]; exists {
			blacklistedListenersSet[listenerName(*listener.Name)] = nil
		}
	}

	// Augment the list of prohibited listeners by looking at the rules
	blacklistedRoutingRules, _ := er.GetBlacklistedRoutingRules()
	for _, rule := range blacklistedRoutingRules {
		if rule.HTTPListener != nil && rule.HTTPListener.ID != nil {
			listenerName := listenerName(utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID))
			blacklistedListenersSet[listenerName] = nil
		}
	}
	return blacklistedListenersSet
}
