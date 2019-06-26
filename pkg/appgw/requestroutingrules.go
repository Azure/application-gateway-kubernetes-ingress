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
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

func (c *appGwConfigBuilder) pathMaps(ingress *v1beta1.Ingress, serviceList []*v1.Service, rule *v1beta1.IngressRule,
	listenerID listenerIdentifier, urlPathMap *n.ApplicationGatewayURLPathMap,
	defaultAddressPoolID string, defaultHTTPSettingsID string) *n.ApplicationGatewayURLPathMap {
	if urlPathMap == nil {
		urlPathMap = &n.ApplicationGatewayURLPathMap{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(generateURLPathMapName(listenerID)),
			ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool:  &n.SubResource{ID: &defaultAddressPoolID},
				DefaultBackendHTTPSettings: &n.SubResource{ID: &defaultHTTPSettingsID},
			},
		}
	}

	if urlPathMap.ApplicationGatewayURLPathMapPropertiesFormat.PathRules == nil {
		urlPathMap.PathRules = &[]n.ApplicationGatewayPathRule{}
	}

	ingressList := c.k8sContext.GetHTTPIngressList()
	backendPools := c.newBackendPoolMap(ingressList, serviceList)
	_, backendHTTPSettingsMap, _, _ := c.getBackendsAndSettingsMap(ingressList, serviceList)
	for pathIdx := range rule.HTTP.Paths {
		path := &rule.HTTP.Paths[pathIdx]
		backendID := generateBackendID(ingress, rule, path, &path.Backend)
		backendPool := backendPools[backendID]
		backendHTTPSettings := backendHTTPSettingsMap[backendID]
		if backendPool == nil || backendHTTPSettings == nil {
			continue
		}
		pathRules := *urlPathMap.PathRules

		backendPoolSubResource := n.SubResource{ID: to.StringPtr(c.appGwIdentifier.addressPoolID(*backendPool.Name))}
		backendHTTPSettingsSubResource := n.SubResource{ID: to.StringPtr(c.appGwIdentifier.httpSettingsID(*backendHTTPSettings.Name))}

		if len(path.Path) == 0 || path.Path == "/*" || path.Path == "/" {
			// this backend should be a default backend, catches all traffic
			// check if it is a host-specific default backend
			if rule.Host == listenerID.HostName {
				// override default backend with host-specific default backend
				urlPathMap.DefaultBackendAddressPool = &backendPoolSubResource
				urlPathMap.DefaultBackendHTTPSettings = &backendHTTPSettingsSubResource
			}
		} else {
			// associate backend with a path-based rule
			pathRules = append(pathRules, n.ApplicationGatewayPathRule{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(generatePathRuleName(ingress.Namespace, ingress.Name, strconv.Itoa(pathIdx))),
				ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
					Paths:               &[]string{path.Path},
					BackendAddressPool:  &backendPoolSubResource,
					BackendHTTPSettings: &backendHTTPSettingsSubResource,
				},
			})
		}

		urlPathMap.PathRules = &pathRules
	}

	return urlPathMap
}

func (c *appGwConfigBuilder) RequestRoutingRules(cbCtx *ConfigBuilderContext) error {
	_, httpListenersMap := c.getListeners(cbCtx)
	urlPathMaps := make(map[listenerIdentifier]*n.ApplicationGatewayURLPathMap)
	backendPools := c.newBackendPoolMap(cbCtx.IngressList, cbCtx.ServiceList)
	_, backendHTTPSettingsMap, _, _ := c.getBackendsAndSettingsMap(cbCtx.IngressList, cbCtx.ServiceList)
	for _, ingress := range cbCtx.IngressList {
		defaultAddressPoolID := c.appGwIdentifier.addressPoolID(defaultBackendAddressPoolName)
		defaultHTTPSettingsID := c.appGwIdentifier.httpSettingsID(defaultBackendHTTPSettingsName)

		var wildcardRule *v1beta1.IngressRule
		wildcardRule = nil
		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			if rule.HTTP != nil && len(rule.Host) == 0 {
				wildcardRule = rule
			}
		}

		// find the default backend for a ingress
		defBackend := ingress.Spec.Backend
		if wildcardRule != nil {
			// wildcard rule override the default backend
			for pathIdx := range wildcardRule.HTTP.Paths {
				path := &wildcardRule.HTTP.Paths[pathIdx]
				if path.Path == "" || path.Path == "/*" || path.Path == "/" {
					// look for default path
					defBackend = &path.Backend
				}
			}
		}

		if defBackend != nil {
			// has default backend
			defaultBackendID := generateBackendID(ingress, nil, nil, defBackend)

			defaultHTTPSettings := backendHTTPSettingsMap[defaultBackendID]
			defaultAddressPool := backendPools[defaultBackendID]
			if defaultAddressPool != nil && defaultHTTPSettings != nil {
				// default settings is valid
				defaultAddressPoolID = c.appGwIdentifier.addressPoolID(*defaultAddressPool.Name)
				defaultHTTPSettingsID = c.appGwIdentifier.httpSettingsID(*defaultHTTPSettings.Name)
			}
		}

		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			if rule.HTTP == nil {
				// skip no http rule
				continue
			}

			listenerHTTPID := generateListenerID(rule, n.HTTP, nil)
			_, httpAvailable := httpListenersMap[listenerHTTPID]

			listenerHTTPSID := generateListenerID(rule, n.HTTPS, nil)
			_, httpsAvailable := httpListenersMap[listenerHTTPSID]

			if httpAvailable {
				if wildcardRule != nil && len(rule.Host) != 0 {
					// only add wildcard rules when host is specified
					urlPathMaps[listenerHTTPID] = c.pathMaps(ingress, cbCtx.ServiceList, wildcardRule,
						listenerHTTPID, urlPathMaps[listenerHTTPID],
						defaultAddressPoolID, defaultHTTPSettingsID)
				}

				// need to eliminate non-unique paths
				urlPathMaps[listenerHTTPID] = c.pathMaps(ingress, cbCtx.ServiceList, rule,
					listenerHTTPID, urlPathMaps[listenerHTTPID],
					defaultAddressPoolID, defaultHTTPSettingsID)

				// If ingress is annotated with "ssl-redirect" and we have TLS - setup redirection configuration.
				if sslRedirect, _ := annotations.IsSslRedirect(ingress); sslRedirect && httpsAvailable {
					c.modifyPathRulesForRedirection(urlPathMaps[listenerHTTPID], listenerHTTPSID)
				}
			}

			if httpsAvailable {
				if wildcardRule != nil && len(rule.Host) != 0 {
					// only add wildcard rules when host is specified
					urlPathMaps[listenerHTTPSID] = c.pathMaps(ingress, cbCtx.ServiceList, wildcardRule,
						listenerHTTPSID, urlPathMaps[listenerHTTPSID],
						defaultAddressPoolID, defaultHTTPSettingsID)
				}

				// need to eliminate non-unique paths
				urlPathMaps[listenerHTTPSID] = c.pathMaps(ingress, cbCtx.ServiceList, rule,
					listenerHTTPSID, urlPathMaps[listenerHTTPSID],
					defaultAddressPoolID, defaultHTTPSettingsID)
			}
		}
	}

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

	var urlPathMapFiltered []n.ApplicationGatewayURLPathMap
	var requestRoutingRules []n.ApplicationGatewayRequestRoutingRule
	for listenerID, urlPathMap := range urlPathMaps {
		httpListener := httpListenersMap[listenerID]
		if len(*urlPathMap.PathRules) == 0 {
			// Basic Rule, because we have no path-based rule
			rule := n.ApplicationGatewayRequestRoutingRule{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(generateRequestRoutingRuleName(listenerID)),
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType:              n.Basic,
					HTTPListener:          &n.SubResource{ID: to.StringPtr(c.appGwIdentifier.listenerID(*httpListener.Name))},
					RedirectConfiguration: urlPathMap.DefaultRedirectConfiguration,
				},
			}

			// We setup the default backend address pools and default backend HTTP settings only if
			// this rule does not have an `ssl-redirect` configuration.
			if rule.RedirectConfiguration == nil {
				rule.BackendAddressPool = urlPathMap.DefaultBackendAddressPool
				rule.BackendHTTPSettings = urlPathMap.DefaultBackendHTTPSettings
			}
			requestRoutingRules = append(requestRoutingRules, rule)
		} else {
			// Path-based Rule
			rule := n.ApplicationGatewayRequestRoutingRule{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(generateRequestRoutingRuleName(listenerID)),
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType:     n.PathBasedRouting,
					HTTPListener: &n.SubResource{ID: to.StringPtr(c.appGwIdentifier.listenerID(*httpListener.Name))},
					URLPathMap:   &n.SubResource{ID: to.StringPtr(c.appGwIdentifier.urlPathMapID(*urlPathMap.Name))},
				},
			}
			urlPathMapFiltered = append(urlPathMapFiltered, *urlPathMap)
			requestRoutingRules = append(requestRoutingRules, rule)
		}
	}

	sort.Sort(sorter.ByRequestRoutingRuleName(requestRoutingRules))
	c.appGw.RequestRoutingRules = &requestRoutingRules

	sort.Sort(sorter.ByPathMap(urlPathMapFiltered))
	c.appGw.URLPathMaps = &urlPathMapFiltered

	return nil
}

func (c *appGwConfigBuilder) getSslRedirectConfigResourceReference(targetListener listenerIdentifier) *n.SubResource {
	configName := generateSSLRedirectConfigurationName(targetListener)
	sslRedirectConfigID := c.appGwIdentifier.redirectConfigurationID(configName)
	return resourceRef(sslRedirectConfigID)
}

func (c *appGwConfigBuilder) modifyPathRulesForRedirection(httpURLPathMap *n.ApplicationGatewayURLPathMap, targetListener listenerIdentifier) {
	// Application Gateway supports Basic and Path-based rules

	if len(*httpURLPathMap.PathRules) == 0 {
		// There are no paths. This is a rule of type "Basic"
		redirectRef := c.getSslRedirectConfigResourceReference(targetListener)
		glog.V(5).Infof("Attaching redirection config %s to basic request routing rule: %s\n", *redirectRef.ID, *httpURLPathMap.Name)

		// URL Path Map must have either DefaultRedirectConfiguration xor (DefaultBackendAddressPool + DefaultBackendHTTPSettings)
		httpURLPathMap.DefaultRedirectConfiguration = redirectRef

		// Since this is a redirect - ensure Default Backend is NOT setup
		httpURLPathMap.DefaultBackendHTTPSettings = nil
		httpURLPathMap.DefaultBackendAddressPool = nil
		return
	}

	for idx := range *httpURLPathMap.PathRules {
		// This is a rule of type "Path-based"
		pathRule := &(*httpURLPathMap.PathRules)[idx]
		redirectRef := c.getSslRedirectConfigResourceReference(targetListener)
		glog.V(5).Infof("Attaching redirection config %s request routing rule: %s\n", *redirectRef.ID, *pathRule.Name)

		// A Path Rule must have either RedirectConfiguration xor (BackendAddressPool + BackendHTTPSettings)
		pathRule.RedirectConfiguration = redirectRef

		// Since this is a redirect - ensure Backend is NOT setup
		pathRule.BackendAddressPool = nil
		pathRule.BackendHTTPSettings = nil
	}
}
