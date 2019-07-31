// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"
	"strconv"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

func (c *appGwConfigBuilder) RequestRoutingRules(cbCtx *ConfigBuilderContext) error {
	requestRoutingRules, pathMaps := c.getRules(cbCtx)

	if cbCtx.EnableBrownfieldDeployment {
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

	if cbCtx.EnableBrownfieldDeployment {
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
	httpListenersMap := c.groupListenersByListenerIdentifier(c.getListeners(cbCtx))
	var pathMap []n.ApplicationGatewayURLPathMap
	var requestRoutingRules []n.ApplicationGatewayRequestRoutingRule
	for listenerID, urlPathMap := range c.getPathMaps(cbCtx) {
		httpListener := httpListenersMap[listenerID]
		rule := n.ApplicationGatewayRequestRoutingRule{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(generateRequestRoutingRuleName(listenerID)),
			ID:   to.StringPtr(c.appGwIdentifier.requestRoutingRuleID(generateRequestRoutingRuleName(listenerID))),
			ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
				HTTPListener: &n.SubResource{ID: to.StringPtr(c.appGwIdentifier.listenerID(*httpListener.Name))},
			},
		}
		glog.V(5).Infof("Binding rule %s to listener %s", *rule.Name, *httpListener.Name)
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
		requestRoutingRules = append(requestRoutingRules, rule)
	}
	return requestRoutingRules, pathMap
}

func (c *appGwConfigBuilder) getPathMaps(cbCtx *ConfigBuilderContext) map[listenerIdentifier]*n.ApplicationGatewayURLPathMap {
	defaultAddressPoolID := to.StringPtr(c.appGwIdentifier.addressPoolID(defaultBackendAddressPoolName))
	defaultHTTPSettingsID := to.StringPtr(c.appGwIdentifier.httpSettingsID(defaultBackendHTTPSettingsName))
	urlPathMaps := make(map[listenerIdentifier]*n.ApplicationGatewayURLPathMap)
	for ingressIdx := range cbCtx.IngressList {
		ingress := cbCtx.IngressList[ingressIdx]
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
							DefaultBackendAddressPool:  &n.SubResource{ID: defaultAddressPoolID},
							DefaultBackendHTTPSettings: &n.SubResource{ID: defaultHTTPSettingsID},
						},
					}
				}

				pathMap := c.getPathMap(cbCtx, listenerID, listenerAzConfig, ingress, rule)
				urlPathMaps[listenerID] = c.mergePathMap(urlPathMaps[listenerID], pathMap)
			}
		}
	}

	for _, virtualService := range cbCtx.IstioVirtualServices {
		for _, rule := range virtualService.Spec.HTTP {
			_, azListenerConfig := c.processIstioIngressRule(&rule, virtualService, cbCtx.EnvVariables)
			for listenerID, listenerAzConfig := range azListenerConfig {
				if _, exists := urlPathMaps[listenerID]; !exists {
					urlPathMaps[listenerID] = &n.ApplicationGatewayURLPathMap{
						Etag: to.StringPtr("*"),
						Name: to.StringPtr(generateURLPathMapName(listenerID)),
						ID:   to.StringPtr(c.appGwIdentifier.urlPathMapID(generateURLPathMapName(listenerID))),
						ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
							DefaultBackendAddressPool:  &n.SubResource{ID: defaultAddressPoolID},
							DefaultBackendHTTPSettings: &n.SubResource{ID: defaultHTTPSettingsID},
						},
					}
				}

				pathMap := c.getIstioPathMap(cbCtx, listenerID, listenerAzConfig, virtualService, &rule)
				urlPathMaps[listenerID] = c.mergePathMap(urlPathMaps[listenerID], pathMap)
			}
		}
	}

	// if no url pathmaps were created, then add a default path map since this will be translated to
	// a basic request routing rule which is needed on Application Gateway to avoid validation error.
	if len(urlPathMaps) == 0 {
		defaultAddressPoolID := c.appGwIdentifier.addressPoolID(defaultBackendAddressPoolName)
		defaultHTTPSettingsID := c.appGwIdentifier.httpSettingsID(defaultBackendHTTPSettingsName)
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

	return urlPathMaps
}

func (c *appGwConfigBuilder) getPathMap(cbCtx *ConfigBuilderContext, listenerID listenerIdentifier, listenerAzConfig listenerAzConfig, ingress *v1beta1.Ingress, rule *v1beta1.IngressRule) *n.ApplicationGatewayURLPathMap {
	// initilize a path map for this listener if doesn't exists
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

func (c *appGwConfigBuilder) getIstioPathMap(cbCtx *ConfigBuilderContext, listenerID listenerIdentifier, listenerAzConfig listenerAzConfig, virtualService *v1alpha3.VirtualService, rule *v1alpha3.HTTPRoute) *n.ApplicationGatewayURLPathMap {
	pathMap := n.ApplicationGatewayURLPathMap{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(generateURLPathMapName(listenerID)),
		ID:   to.StringPtr(c.appGwIdentifier.urlPathMapID(generateURLPathMapName(listenerID))),
		ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{},
	}

	/* TODO(rhea): add defaults and path rules */
	return &pathMap
}

func (c *appGwConfigBuilder) getDefaultFromRule(cbCtx *ConfigBuilderContext, listenerID listenerIdentifier, listenerAzConfig listenerAzConfig, ingress *v1beta1.Ingress, rule *v1beta1.IngressRule) (*string, *string, *string) {
	var defaultAddressPoolID *string
	var defaultHTTPSettingsID *string
	var defaultRedirectConfigurationID *string

	if sslRedirect, _ := annotations.IsSslRedirect(ingress); sslRedirect && listenerAzConfig.Protocol == n.HTTP {
		redirectName := generateSSLRedirectConfigurationName(listenerIdentifier{HostName: listenerID.HostName, FrontendPort: 443, UsePrivateIP: listenerID.UsePrivateIP})
		defaultRedirectConfigurationID = to.StringPtr(c.appGwIdentifier.redirectConfigurationID(redirectName))
		return nil, nil, defaultRedirectConfigurationID
	}

	var defRule *v1beta1.IngressRule
	var defPath *v1beta1.HTTPIngressPath
	defBackend := ingress.Spec.Backend
	for pathIdx := range rule.HTTP.Paths {
		path := rule.HTTP.Paths[pathIdx]
		if path.Path == "" || path.Path == "/*" || path.Path == "/" {
			defBackend = &path.Backend
			defPath = &path
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
			defaultAddressPoolID = to.StringPtr(c.appGwIdentifier.addressPoolID(*defaultAddressPool.Name))
			defaultHTTPSettingsID = to.StringPtr(c.appGwIdentifier.httpSettingsID(*defaultHTTPSettings.Name))
		}
	}

	return defaultAddressPoolID, defaultHTTPSettingsID, nil
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
			redirectName := generateSSLRedirectConfigurationName(listenerIdentifier{HostName: listenerID.HostName, FrontendPort: 443, UsePrivateIP: listenerID.UsePrivateIP})
			redirectID := c.appGwIdentifier.redirectConfigurationID(redirectName)
			pathRule.RedirectConfiguration = resourceRef(redirectID)
			glog.V(5).Infof("Attaching redirection %s to path rule: %s", redirectName, *pathRule.Name)
		} else {
			backendID := generateBackendID(ingress, rule, path, &path.Backend)
			backendPool := backendPools[backendID]
			backendHTTPSettings := backendHTTPSettingsMap[backendID]
			if backendPool == nil || backendHTTPSettings == nil {
				continue
			}

			pathRule.BackendAddressPool = &n.SubResource{ID: backendPool.ID}
			pathRule.BackendHTTPSettings = &n.SubResource{ID: backendHTTPSettings.ID}
			glog.V(5).Infof("Attaching pool %s and http setting %s to path rule: %s", *backendPool.Name, *backendHTTPSettings.Name, *pathRule.Name)
		}

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
