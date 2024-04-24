// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

// CleanUpPathRulesAddedByAGIC removes path rules that are created by AGIC
func (c *appGwConfigBuilder) CleanUpPathRulesAddedByAGIC() {
	pathRuleNamePrefix := fmt.Sprintf("%s%s-", agPrefix, prefixPathRule)

	// Remove path rules that are created by AGIC
	for _, pathMap := range *c.appGw.URLPathMaps {
		var pathRulesAddedManually []n.ApplicationGatewayPathRule
		for _, pathRule := range *pathMap.PathRules {
			if !strings.HasPrefix(*pathRule.Name, pathRuleNamePrefix) {
				pathRulesAddedManually = append(pathRulesAddedManually, pathRule)
			}
		}

		pathMap.PathRules = &pathRulesAddedManually
	}
}

// CleanUpUnusedDefaults removes the default backend and default http settings if they are not used by any ingress
func (c *appGwConfigBuilder) CleanUpUnusedDefaults() {
	if !c.isPoolUsed(DefaultBackendAddressPoolName) {
		c.removePool(DefaultBackendAddressPoolName)
	}

	if !c.isBackendSettingsUsed(DefaultBackendHTTPSettingsName) {
		c.removeBackendSettings(DefaultBackendHTTPSettingsName)
	}

	if !c.isProbeUsed(defaultProbeName(n.ApplicationGatewayProtocolHTTP)) {
		c.removeProbe(defaultProbeName(n.ApplicationGatewayProtocolHTTP))
	}

	if !c.isProbeUsed(defaultProbeName(n.ApplicationGatewayProtocolHTTPS)) {
		c.removeProbe(defaultProbeName(n.ApplicationGatewayProtocolHTTPS))
	}
}

func (c *appGwConfigBuilder) isPoolUsed(name string) bool {
	isDefaultRef := func(ref *n.SubResource) bool {
		return ref != nil &&
			ref.ID != nil &&
			resourceIDHasResourceName(*ref.ID, name)
	}

	for _, i := range *c.appGw.RequestRoutingRules {
		if isDefaultRef(i.BackendAddressPool) {
			return true
		}
	}

	for _, i := range *c.appGw.URLPathMaps {
		if isDefaultRef(i.DefaultBackendAddressPool) {
			return true
		}

		for _, p := range *i.PathRules {
			if isDefaultRef(p.BackendAddressPool) {
				return true
			}
		}
	}

	return false
}

func (c *appGwConfigBuilder) removePool(name string) {
	pools := *c.appGw.BackendAddressPools
	for bIdx, i := range pools {
		if resourceIDHasResourceName(*i.ID, name) {
			pools = append(pools[:bIdx], pools[bIdx+1:]...)
			break
		}
	}
	c.appGw.BackendAddressPools = &pools
}

func (c *appGwConfigBuilder) isBackendSettingsUsed(name string) bool {
	isDefaultRef := func(ref *n.SubResource) bool {
		return ref != nil &&
			ref.ID != nil &&
			resourceIDHasResourceName(*ref.ID, name)
	}

	for _, i := range *c.appGw.RequestRoutingRules {
		if isDefaultRef(i.BackendHTTPSettings) {
			return true
		}
	}

	for _, i := range *c.appGw.URLPathMaps {
		if isDefaultRef(i.DefaultBackendHTTPSettings) {
			return true
		}

		for _, p := range *i.PathRules {
			if isDefaultRef(p.BackendHTTPSettings) {
				return true
			}
		}
	}

	return false
}

func (c *appGwConfigBuilder) removeBackendSettings(name string) {
	settings := *c.appGw.BackendHTTPSettingsCollection
	for bIdx, i := range settings {
		if resourceIDHasResourceName(*i.ID, name) {
			settings = append(settings[:bIdx], settings[bIdx+1:]...)
			break
		}
	}
	c.appGw.BackendHTTPSettingsCollection = &settings
}

func (c *appGwConfigBuilder) isProbeUsed(name string) bool {
	isDefaultRef := func(ref *n.SubResource) bool {
		return ref != nil &&
			ref.ID != nil &&
			resourceIDHasResourceName(*ref.ID, name)
	}

	for _, i := range *c.appGw.BackendHTTPSettingsCollection {
		if isDefaultRef(i.Probe) {
			return true
		}
	}

	return false
}

func (c *appGwConfigBuilder) removeProbe(name string) {
	probes := *c.appGw.Probes
	for bIdx, i := range probes {
		if resourceIDHasResourceName(*i.ID, name) {
			probes = append(probes[:bIdx], probes[bIdx+1:]...)
			break
		}
	}
	c.appGw.Probes = &probes
}

func resourceIDHasResourceName(resourceID string, resourceNameToMatch string) bool {
	splits := strings.Split(resourceID, "/")
	if len(splits) == 0 {
		return false
	}

	resourceName := splits[len(splits)-1]
	return strings.EqualFold(resourceName, resourceNameToMatch)
}
