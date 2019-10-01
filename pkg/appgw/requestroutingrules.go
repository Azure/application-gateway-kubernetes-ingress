// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"
	"strconv"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

func (c *appGwConfigBuilder) RequestRoutingRules(cbCtx *ConfigBuilderContext) error {
	requestRoutingRules, pathMaps := c.getRules(cbCtx)

	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		rCtx := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)
		{
			// PathMaps we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
			existingBlacklisted, existingNonBlacklisted := rCtx.GetBlacklistedPathMaps()

			brownfield.LogPathMaps(existingBlacklisted, existingNonBlacklisted, pathMaps)

			// MergePathMaps would produce unique list of routing rules based on Name. Routing rules, which have the same name
			// as a managed rule would be overwritten.
			pathMaps = brownfield.MergePathMaps(existingBlacklisted, pathMaps)
		}
	}

	sort.Sort(sorter.ByPathMap(pathMaps))
	c.appGw.URLPathMaps = &pathMaps

	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		rCtx := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)
		{
			// RoutingRules we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
			existingBlacklisted, existingNonBlacklisted := rCtx.GetBlacklistedRoutingRules()

			brownfield.LogRules(existingBlacklisted, existingNonBlacklisted, requestRoutingRules)

			// MergeRules would produce unique list of routing rules based on Name. Routing rules, which have the same name
			// as a managed rule would be overwritten.
			requestRoutingRules = brownfield.MergeRules(&c.appGw, existingBlacklisted, requestRoutingRules)
		}
	}

	sort.Sort(sorter.ByRequestRoutingRuleName(requestRoutingRules))
	c.appGw.RequestRoutingRules = &requestRoutingRules

	return nil
}

func (c *appGwConfigBuilder) getRules(cbCtx *ConfigBuilderContext) ([]n.ApplicationGatewayRequestRoutingRule, []n.ApplicationGatewayURLPathMap) {
	if c.mem.routingRules != nil && c.mem.pathMaps != nil {
		return *c.mem.routingRules, *c.mem.pathMaps
	}
	httpListenersMap := c.groupListenersByListenerIdentifier(cbCtx)
	var pathMap []n.ApplicationGatewayURLPathMap
	var requestRoutingRules []n.ApplicationGatewayRequestRoutingRule
	for listenerID, urlPathMap := range c.getPathMaps(cbCtx) {
		routingRuleName := generateRequestRoutingRuleName(listenerID)
		httpListener, exists := httpListenersMap[listenerID]
		if !exists {
			glog.Errorf("Routing rule %s will not be created; listener %+v does not exist", routingRuleName, listenerID)
			continue
		}
		rule := n.ApplicationGatewayRequestRoutingRule{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(routingRuleName),
			ID:   to.StringPtr(c.appGwIdentifier.requestRoutingRuleID(generateRequestRoutingRuleName(listenerID))),
			ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
				HTTPListener: &n.SubResource{ID: to.StringPtr(c.appGwIdentifier.listenerID(*httpListener.Name))},
			},
		}
		if urlPathMap.PathRules == nil || len(*urlPathMap.PathRules) == 0 {
			// Basic Rule, because we have no path-based rule
			rule.RuleType = n.Basic
			rule.RedirectConfiguration = urlPathMap.DefaultRedirectConfiguration

			// We setup the default backend address pools and default backend HTTP settings only if
			// this rule does not have an `ssl-redirect` configuration.
			if rule.RedirectConfiguration == nil {
				rule.BackendAddressPool = urlPathMap.DefaultBackendAddressPool
				rule.BackendHTTPSettings = urlPathMap.DefaultBackendHTTPSettings
			}
		} else {
			// Path-based Rule
			rule.RuleType = n.PathBasedRouting
			rule.URLPathMap = &n.SubResource{ID: to.StringPtr(c.appGwIdentifier.urlPathMapID(*urlPathMap.Name))}
			pathMap = append(pathMap, *urlPathMap)
		}
		if rule.RuleType == n.PathBasedRouting {
			glog.V(5).Infof("Bound path-based rule %s to listener %s(%s, %d) and url path map %s", *rule.Name, *httpListener.Name, listenerID.HostName, listenerID.FrontendPort, utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID))
		} else {
			if rule.RedirectConfiguration != nil {
				glog.V(5).Infof("Bound basic rule %s to listener %s(%s, %d) and redirect configuration %s", *rule.Name, *httpListener.Name, listenerID.HostName, listenerID.FrontendPort, utils.GetLastChunkOfSlashed(*rule.RedirectConfiguration.ID))
			} else {
				glog.V(5).Infof("Bound basic rule %s to listener %s(%s, %d), backend pool %s and backend http settings %s", *rule.Name, *httpListener.Name, listenerID.HostName, listenerID.FrontendPort, utils.GetLastChunkOfSlashed(*rule.BackendAddressPool.ID), utils.GetLastChunkOfSlashed(*rule.BackendHTTPSettings.ID))
			}
		}
		requestRoutingRules = append(requestRoutingRules, rule)
	}

	c.mem.routingRules = &requestRoutingRules
	c.mem.pathMaps = &pathMap
	return requestRoutingRules, pathMap
}

func (c *appGwConfigBuilder) noRulesIngress(cbCtx *ConfigBuilderContext, ingress *v1beta1.Ingress, urlPathMaps *map[listenerIdentifier]*n.ApplicationGatewayURLPathMap) {
	// There are no Rules. We are dealing with some very rudimentary Ingress definition.
	if ingress.Spec.Backend == nil {
		return;
	}
	backendID := generateBackendID(ingress, nil, nil, ingress.Spec.Backend)
	_, _, serviceBackendPairMap, err := c.getBackendsAndSettingsMap(cbCtx)
	if err != nil {
		glog.Error("Error fetching Backends and Settings: ", err)
	}
	if serviceBackendPair, exists := serviceBackendPairMap[backendID]; exists {
		poolName := generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), serviceBackendPair.BackendPort)
		defaultAddressPoolID := c.appGwIdentifier.AddressPoolID(poolName)
		defaultHTTPSettingsID := c.appGwIdentifier.HTTPSettingsID(DefaultBackendHTTPSettingsName)
		listenerID := defaultFrontendListenerIdentifier()
		(*urlPathMaps)[listenerID] = &n.ApplicationGatewayURLPathMap{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(generateURLPathMapName(listenerID)),
			ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool:  &n.SubResource{ID: &defaultAddressPoolID},
				DefaultBackendHTTPSettings: &n.SubResource{ID: &defaultHTTPSettingsID},
				PathRules:                  &[]n.ApplicationGatewayPathRule{},
			},
		}
	}
}

func (c *appGwConfigBuilder) getPathMaps(cbCtx *ConfigBuilderContext) map[listenerIdentifier]*n.ApplicationGatewayURLPathMap {
	urlPathMaps := make(map[listenerIdentifier]*n.ApplicationGatewayURLPathMap)
	for ingressIdx := range cbCtx.IngressList {
		ingress := cbCtx.IngressList[ingressIdx]

		if len(ingress.Spec.Rules) == 0 {
			c.noRulesIngress(cbCtx, ingress, &urlPathMaps)
		}

		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			// skip no http rule
			if rule.HTTP == nil {
				continue
			}

			_, azListenerConfig := c.processIngressRule(rule, ingress, cbCtx.EnvVariables)
			for listenerID, listenerAzConfig := range azListenerConfig {
				if _, exists := urlPathMaps[listenerID]; !exists {
					urlPathMaps[listenerID] = &n.ApplicationGatewayURLPathMap{
						Etag: to.StringPtr("*"),
						Name: to.StringPtr(generateURLPathMapName(listenerID)),
						ID:   to.StringPtr(c.appGwIdentifier.urlPathMapID(generateURLPathMapName(listenerID))),
						ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
							DefaultBackendAddressPool:  &n.SubResource{ID: cbCtx.DefaultAddressPoolID},
							DefaultBackendHTTPSettings: &n.SubResource{ID: cbCtx.DefaultHTTPSettingsID},
						},
					}
				}

				pathMap := c.getPathMap(cbCtx, listenerID, listenerAzConfig, ingress, rule)
				urlPathMaps[listenerID] = c.mergePathMap(urlPathMaps[listenerID], pathMap)
			}
		}
	}

	// if no url pathmaps were created, then add a default path map since this will be translated to
	// a basic request routing rule which is needed on Application Gateway to avoid validation error.
	if len(urlPathMaps) == 0 {
		defaultAddressPoolID := c.appGwIdentifier.AddressPoolID(DefaultBackendAddressPoolName)
		defaultHTTPSettingsID := c.appGwIdentifier.HTTPSettingsID(DefaultBackendHTTPSettingsName)
		listenerID := defaultFrontendListenerIdentifier()
		urlPathMaps[listenerID] = &n.ApplicationGatewayURLPathMap{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(generateURLPathMapName(listenerID)),
			ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool:  &n.SubResource{ID: &defaultAddressPoolID},
				DefaultBackendHTTPSettings: &n.SubResource{ID: &defaultHTTPSettingsID},
				PathRules:                  &[]n.ApplicationGatewayPathRule{},
			},
		}
	}

	if cbCtx.EnvVariables.EnableIstioIntegration {
		for listenerID, pathMap := range c.getIstioPathMaps(cbCtx) {
			if _, exists := urlPathMaps[listenerID]; !exists {
				urlPathMaps[listenerID] = pathMap
			}
		}
	}

	return urlPathMaps
}

func (c *appGwConfigBuilder) getPathMap(cbCtx *ConfigBuilderContext, listenerID listenerIdentifier, listenerAzConfig listenerAzConfig, ingress *v1beta1.Ingress, rule *v1beta1.IngressRule) *n.ApplicationGatewayURLPathMap {
	// initialize a path map for this listener if doesn't exists
	pathMap := n.ApplicationGatewayURLPathMap{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(generateURLPathMapName(listenerID)),
		ID:   to.StringPtr(c.appGwIdentifier.urlPathMapID(generateURLPathMapName(listenerID))),
		ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{},
	}

	// get defaults provided by the rules if any
	defaultAddressPoolID, defaultHTTPSettingsID, defaultRedirectConfigurationID := c.getDefaultFromRule(cbCtx, listenerID, listenerAzConfig, ingress, rule)
	if defaultRedirectConfigurationID != nil {
		pathMap.DefaultRedirectConfiguration = resourceRef(*defaultRedirectConfigurationID)
		pathMap.DefaultBackendAddressPool = nil
		pathMap.DefaultBackendHTTPSettings = nil
	} else if defaultAddressPoolID != nil && defaultHTTPSettingsID != nil {
		pathMap.DefaultBackendAddressPool = resourceRef(*defaultAddressPoolID)
		pathMap.DefaultBackendHTTPSettings = resourceRef(*defaultHTTPSettingsID)
	}

	pathMap.PathRules = c.getPathRules(cbCtx, listenerID, listenerAzConfig, ingress, rule)

	return &pathMap
}

func (c *appGwConfigBuilder) getDefaultFromRule(cbCtx *ConfigBuilderContext, listenerID listenerIdentifier, listenerAzConfig listenerAzConfig, ingress *v1beta1.Ingress, rule *v1beta1.IngressRule) (*string, *string, *string) {
	if sslRedirect, _ := annotations.IsSslRedirect(ingress); sslRedirect && listenerAzConfig.Protocol == n.HTTP {
		targetListener := listenerIdentifier{
			HostName:     listenerID.HostName,
			FrontendPort: 443,
			UsePrivateIP: listenerID.UsePrivateIP,
		}

		// We could end up in a situation where we are attempting to attach a redirect, which does not exist.
		redirectRef := c.getSslRedirectConfigResourceReference(targetListener)
		redirectsSet := *c.groupRedirectsByID(c.getRedirectConfigurations(cbCtx))

		if _, exists := redirectsSet[*redirectRef.ID]; exists {
			glog.V(5).Infof("Attached default redirection %s to rule %+v", *redirectRef.ID, *rule)
			return nil, nil, redirectRef.ID
		}
		glog.Errorf("Will not attach default redirect to rule; SSL Redirect does not exist: %s", *redirectRef.ID)
	}

	var defRule *v1beta1.IngressRule
	var defPath *v1beta1.HTTPIngressPath
	defBackend := ingress.Spec.Backend
	for pathIdx := range rule.HTTP.Paths {
		path := &rule.HTTP.Paths[pathIdx]
		if path.Path == "" || path.Path == "/*" || path.Path == "/" {
			defBackend = &path.Backend
			defPath = path
			defRule = rule
		}
	}

	backendPools := c.newBackendPoolMap(cbCtx)
	_, backendHTTPSettingsMap, _, _ := c.getBackendsAndSettingsMap(cbCtx)
	if defBackend != nil {
		// has default backend
		defaultBackendID := generateBackendID(ingress, defRule, defPath, defBackend)
		defaultHTTPSettings := backendHTTPSettingsMap[defaultBackendID]
		defaultAddressPool := backendPools[defaultBackendID]
		if defaultAddressPool != nil && defaultHTTPSettings != nil {
			poolID := to.StringPtr(c.appGwIdentifier.AddressPoolID(*defaultAddressPool.Name))
			settID := to.StringPtr(c.appGwIdentifier.HTTPSettingsID(*defaultHTTPSettings.Name))
			return poolID, settID, nil
		}
	}

	return cbCtx.DefaultAddressPoolID, cbCtx.DefaultHTTPSettingsID, nil
}

func (c *appGwConfigBuilder) getPathRules(cbCtx *ConfigBuilderContext, listenerID listenerIdentifier, listenerAzConfig listenerAzConfig, ingress *v1beta1.Ingress, rule *v1beta1.IngressRule) *[]n.ApplicationGatewayPathRule {
	backendPools := c.newBackendPoolMap(cbCtx)
	_, backendHTTPSettingsMap, _, _ := c.getBackendsAndSettingsMap(cbCtx)
	pathRules := make([]n.ApplicationGatewayPathRule, 0)
	for pathIdx := range rule.HTTP.Paths {
		path := &rule.HTTP.Paths[pathIdx]
		if len(path.Path) == 0 || path.Path == "/*" || path.Path == "/" {
			continue
		}

		pathRule := n.ApplicationGatewayPathRule{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(generatePathRuleName(ingress.Namespace, ingress.Name, strconv.Itoa(pathIdx))),
			ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
				Paths: &[]string{path.Path},
			},
		}

		if sslRedirect, _ := annotations.IsSslRedirect(ingress); sslRedirect && listenerAzConfig.Protocol == n.HTTP {
			targetListener := listenerIdentifier{
				HostName:     listenerID.HostName,
				FrontendPort: 443,
				UsePrivateIP: listenerID.UsePrivateIP,
			}

			// We could end up in a situation where we are attempting to attach a redirect, which does not exist.
			redirectRef := c.getSslRedirectConfigResourceReference(targetListener)
			redirectsSet := *c.groupRedirectsByID(c.getRedirectConfigurations(cbCtx))

			if _, exists := redirectsSet[*redirectRef.ID]; exists {
				// This Path Rule has a SSL Redirect!
				// Add it and move on to the next Path Rule; No need to attach Backend Pools and Settings
				pathRule.RedirectConfiguration = redirectRef
				glog.V(5).Infof("Attached redirection %s to path rule: %s", *redirectRef.ID, *pathRule.Name)
				pathRules = append(pathRules, pathRule)
				continue
			} else {
				glog.Errorf("Will not attach redirect to rule; SSL Redirect does not exist: %s", *redirectRef.ID)
			}

		}
		backendID := generateBackendID(ingress, rule, path, &path.Backend)
		backendPool := backendPools[backendID]
		backendHTTPSettings := backendHTTPSettingsMap[backendID]
		if backendPool == nil || backendHTTPSettings == nil {
			continue
		}

		pathRule.BackendAddressPool = &n.SubResource{ID: backendPool.ID}
		pathRule.BackendHTTPSettings = &n.SubResource{ID: backendHTTPSettings.ID}
		glog.V(5).Infof("Attached pool %s and http setting %s to path rule: %s", *backendPool.Name, *backendHTTPSettings.Name, *pathRule.Name)

		pathRules = append(pathRules, pathRule)
	}

	return &pathRules
}

func (c *appGwConfigBuilder) mergePathMap(existingPathMap *n.ApplicationGatewayURLPathMap, pathMapToMerge *n.ApplicationGatewayURLPathMap) *n.ApplicationGatewayURLPathMap {
	if pathMapToMerge.DefaultBackendAddressPool != nil {
		existingPathMap.DefaultBackendAddressPool = pathMapToMerge.DefaultBackendAddressPool
	}
	if pathMapToMerge.DefaultBackendHTTPSettings != nil {
		existingPathMap.DefaultBackendHTTPSettings = pathMapToMerge.DefaultBackendHTTPSettings
	}
	if pathMapToMerge.DefaultRedirectConfiguration != nil {
		existingPathMap.DefaultRedirectConfiguration = pathMapToMerge.DefaultRedirectConfiguration
		existingPathMap.DefaultBackendAddressPool = nil
		existingPathMap.DefaultBackendHTTPSettings = nil
	}
	if pathMapToMerge.PathRules == nil || len(*pathMapToMerge.PathRules) == 0 {
		return existingPathMap
	}

	if existingPathMap.PathRules == nil {
		existingPathMap.PathRules = pathMapToMerge.PathRules
	} else {
		pathRules := append(*existingPathMap.PathRules, *pathMapToMerge.PathRules...)
		existingPathMap.PathRules = &pathRules
	}
	return existingPathMap
}
