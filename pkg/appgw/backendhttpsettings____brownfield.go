// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
)

func (c *appGwConfigBuilder) getExistingUnmanagedSettings(cbCtx *ConfigBuilderContext) (*[]n.ApplicationGatewayBackendHTTPSettings, error) {

	newHTTPSettings, _, _, err := c.getBackendsAndSettingsMap(cbCtx)
	if err != nil {
		// NOW WHAT?
	}
	routingRules, _ := c.getRules(cbCtx)
	listenersByName := c.getListenersByName(cbCtx)
	pathMapsByName := c.getPathsByName(cbCtx)

	managed, err := brownfield.GetManagedSettings(newHTTPSettings, routingRules, listenersByName, pathMapsByName, cbCtx.ManagedTargets, cbCtx.ProhibitedTargets)
	if err != nil {
		return nil, err
	}
	managedMap := make(map[string]n.ApplicationGatewayBackendHTTPSettings)
	for _, s := range *managed {
		managedMap[*s.Name] = s
	}
	var unmanagedSettings []n.ApplicationGatewayBackendHTTPSettings

	if c.appGw.BackendHTTPSettingsCollection == nil {
		return &unmanagedSettings, nil
	}

	for _, s := range *c.appGw.BackendHTTPSettingsCollection {
		if _, isManaged := managedMap[*s.Name]; !isManaged {
			unmanagedSettings = append(unmanagedSettings, s)
		}
	}
	return &unmanagedSettings, nil
}

func (c appGwConfigBuilder) getListenersByName(cbCtx *ConfigBuilderContext) brownfield.ListenersByName {
	listeners := make(map[string]*n.ApplicationGatewayHTTPListener)
	_, listenerMap := c.getListeners(cbCtx)
	for _, listener := range listenerMap {
		listeners[*listener.Name] = listener
	}
	return listeners
}

func (c appGwConfigBuilder) getPathsByName(cbCtx *ConfigBuilderContext) brownfield.URLPathMapByName {
	_, paths := c.getRules(cbCtx)
	pathMap := make(map[string]n.ApplicationGatewayURLPathMap)
	for _, path := range paths {
		pathMap[*path.Name] = path
	}
	return pathMap
}

func mergeSettings(s1 *[]n.ApplicationGatewayBackendHTTPSettings, s2 *[]n.ApplicationGatewayBackendHTTPSettings) *[]n.ApplicationGatewayBackendHTTPSettings {
	var merged []n.ApplicationGatewayBackendHTTPSettings
	for _, s := range *s1 {
		merged = append(merged, s)
	}
	for _, s := range *s2 {
		merged = append(merged, s)
	}
	return &merged
}
