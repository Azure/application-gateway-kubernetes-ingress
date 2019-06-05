// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"
	"strconv"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
)

func (c *appGwConfigBuilder) pathMaps(ingress *v1beta1.Ingress, rule *v1beta1.IngressRule,
	listenerID listenerIdentifier, urlPathMap *network.ApplicationGatewayURLPathMap,
	defaultAddressPoolID string, defaultHTTPSettingsID string) *network.ApplicationGatewayURLPathMap {
	if urlPathMap == nil {
		urlPathMapName := generateURLPathMapName(listenerID)
		urlPathMap = &network.ApplicationGatewayURLPathMap{
			Etag: to.StringPtr("*"),
			Name: &urlPathMapName,
			ApplicationGatewayURLPathMapPropertiesFormat: &network.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool:  &network.SubResource{ID: &defaultAddressPoolID},
				DefaultBackendHTTPSettings: &network.SubResource{ID: &defaultHTTPSettingsID},
			},
		}
	}

	if urlPathMap.ApplicationGatewayURLPathMapPropertiesFormat.PathRules == nil {
		urlPathMap.PathRules = &[]network.ApplicationGatewayPathRule{}
	}

	for pathIdx := range rule.HTTP.Paths {
		path := &rule.HTTP.Paths[pathIdx]
		backendID := generateBackendID(ingress, rule, path, &path.Backend)
		backendPool := c.backendPoolMap[backendID]
		backendHTTPSettings := c.backendHTTPSettingsMap[backendID]
		if backendPool == nil || backendHTTPSettings == nil {
			continue
		}
		pathRules := *urlPathMap.PathRules

		backendPoolSubResource := network.SubResource{ID: to.StringPtr(c.appGwIdentifier.addressPoolID(*backendPool.Name))}
		backendHTTPSettingsSubResource := network.SubResource{ID: to.StringPtr(c.appGwIdentifier.httpSettingsID(*backendHTTPSettings.Name))}

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
			pathName := "k8s-" + strconv.Itoa(len(pathRules))
			pathRules = append(pathRules, network.ApplicationGatewayPathRule{
				Etag: to.StringPtr("*"),
				Name: &pathName,
				ApplicationGatewayPathRulePropertiesFormat: &network.ApplicationGatewayPathRulePropertiesFormat{
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

func (c *appGwConfigBuilder) RequestRoutingRules(ingressList []*v1beta1.Ingress) error {
	_, httpListenersMap := c.getListeners(ingressList)
	urlPathMaps := make(map[listenerIdentifier]*network.ApplicationGatewayURLPathMap)
	for _, ingress := range ingressList {
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

			defaultHTTPSettings := c.backendHTTPSettingsMap[defaultBackendID]
			defaultAddressPool := c.backendPoolMap[defaultBackendID]
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

			httpAvailable := false
			httpsAvailable := false

			listenerHTTPID := generateListenerID(rule, network.HTTP, nil)
			if _, exist := httpListenersMap[listenerHTTPID]; exist {
				httpAvailable = true
			}

			// check annotation for port override
			listenerHTTPSID := generateListenerID(rule, network.HTTPS, nil)
			if _, exist := httpListenersMap[listenerHTTPSID]; exist {
				httpsAvailable = true
			}

			if httpAvailable {
				if wildcardRule != nil && len(rule.Host) != 0 {
					// only add wildcard rules when host is specified
					urlPathMaps[listenerHTTPID] = c.pathMaps(ingress, wildcardRule,
						listenerHTTPID, urlPathMaps[listenerHTTPID],
						defaultAddressPoolID, defaultHTTPSettingsID)
				}

				// need to eliminate non-unique paths
				urlPathMaps[listenerHTTPID] = c.pathMaps(ingress, rule,
					listenerHTTPID, urlPathMaps[listenerHTTPID],
					defaultAddressPoolID, defaultHTTPSettingsID)

				// If ingress is annotated with "ssl-redirect" and we have TLS - setup redirection configuration.
				if sslRedirect, _ := annotations.IsSslRedirect(ingress); sslRedirect && httpsAvailable {
					c.modifyPathRulesForRedirection(ingress, urlPathMaps[listenerHTTPID])
				}
			}

			if httpsAvailable {
				if wildcardRule != nil && len(rule.Host) != 0 {
					// only add wildcard rules when host is specified
					urlPathMaps[listenerHTTPSID] = c.pathMaps(ingress, wildcardRule,
						listenerHTTPSID, urlPathMaps[listenerHTTPSID],
						defaultAddressPoolID, defaultHTTPSettingsID)
				}

				// need to eliminate non-unique paths
				urlPathMaps[listenerHTTPSID] = c.pathMaps(ingress, rule,
					listenerHTTPSID, urlPathMaps[listenerHTTPSID],
					defaultAddressPoolID, defaultHTTPSettingsID)
			}
		}
	}

	if len(urlPathMaps) == 0 {
		defaultAddressPoolID := c.appGwIdentifier.addressPoolID(defaultBackendAddressPoolName)
		defaultHTTPSettingsID := c.appGwIdentifier.httpSettingsID(defaultBackendHTTPSettingsName)
		listenerID := defaultFrontendListenerIdentifier()
		urlPathMapName := generateURLPathMapName(listenerID)
		urlPathMaps[listenerID] = &network.ApplicationGatewayURLPathMap{
			Etag: to.StringPtr("*"),
			Name: &urlPathMapName,
			ApplicationGatewayURLPathMapPropertiesFormat: &network.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool:  &network.SubResource{ID: &defaultAddressPoolID},
				DefaultBackendHTTPSettings: &network.SubResource{ID: &defaultHTTPSettingsID},
				PathRules:                  &[]network.ApplicationGatewayPathRule{},
			},
		}
	}

	var urlPathMapFiltered []network.ApplicationGatewayURLPathMap
	var requestRoutingRules []network.ApplicationGatewayRequestRoutingRule
	for listenerID, urlPathMap := range urlPathMaps {
		requestRoutingRuleName := generateRequestRoutingRuleName(listenerID)
		httpListener := httpListenersMap[listenerID]
		httpListenerSubResource := network.SubResource{ID: to.StringPtr(c.appGwIdentifier.listenerID(*httpListener.Name))}
		var rule network.ApplicationGatewayRequestRoutingRule
		if len(*urlPathMap.PathRules) == 0 {
			// Basic Rule, because we have no path-based rule
			rule = network.ApplicationGatewayRequestRoutingRule{
				Etag: to.StringPtr("*"),
				Name: &requestRoutingRuleName,
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &network.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType:              network.Basic,
					HTTPListener:          &httpListenerSubResource,
					RedirectConfiguration: urlPathMap.DefaultRedirectConfiguration,
				},
			}

			// We setup the default backend address pools and default backend HTTP settings only if
			// this rule does not have an `ssl-redirect` configuration.
			if rule.RedirectConfiguration == nil {
				rule.BackendAddressPool = urlPathMap.DefaultBackendAddressPool
				rule.BackendHTTPSettings = urlPathMap.DefaultBackendHTTPSettings
			}
		} else {
			// Path-based Rule
			urlPathMapSubResource := network.SubResource{ID: to.StringPtr(c.appGwIdentifier.urlPathMapID(*urlPathMap.Name))}
			rule = network.ApplicationGatewayRequestRoutingRule{
				Etag: to.StringPtr("*"),
				Name: &requestRoutingRuleName,
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &network.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType:     network.PathBasedRouting,
					HTTPListener: &httpListenerSubResource,
					URLPathMap:   &urlPathMapSubResource,
				},
			}
			urlPathMapFiltered = append(urlPathMapFiltered, *urlPathMap)
		}
		if len(*httpListener.HostName) == 0 {
			requestRoutingRules = append(requestRoutingRules, rule)
		} else {
			requestRoutingRules = append([]network.ApplicationGatewayRequestRoutingRule{rule},
				requestRoutingRules...)
		}
	}

	sort.Sort(sorter.ByRequestRoutingRuleName(requestRoutingRules))
	c.appGwConfig.RequestRoutingRules = &requestRoutingRules

	sort.Sort(sorter.ByRequestRoutingRuleName(requestRoutingRules))
	c.appGwConfig.URLPathMaps = &urlPathMapFiltered

	return nil
}

func (c *appGwConfigBuilder) getSslRedirectConfigResourceReference(ingress *v1beta1.Ingress) *network.SubResource {
	configName := generateSSLRedirectConfigurationName(ingress.Namespace, ingress.Name)
	sslRedirectConfigID := c.appGwIdentifier.redirectConfigurationID(configName)
	return resourceRef(sslRedirectConfigID)
}

func (c *appGwConfigBuilder) modifyPathRulesForRedirection(ingress *v1beta1.Ingress, httpURLPathMap *network.ApplicationGatewayURLPathMap) {
	// Application Gateway supports Basic and Path-based rules

	if len(*httpURLPathMap.PathRules) == 0 {
		// There are no paths. This is a rule of type "Basic"
		redirectRef := c.getSslRedirectConfigResourceReference(ingress)
		glog.Infof("Attaching redirection config %s to basic request routing rule: %s\n", *redirectRef.ID, *httpURLPathMap.Name)

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
		redirectRef := c.getSslRedirectConfigResourceReference(ingress)
		glog.Infof("Attaching redirection config %s request routing rule: %s\n", *redirectRef.ID, *pathRule.Name)

		// A Path Rule must have either RedirectConfiguration xor (BackendAddressPool + BackendHTTPSettings)
		pathRule.RedirectConfiguration = redirectRef

		// Since this is a redirect - ensure Backend is NOT setup
		pathRule.BackendAddressPool = nil
		pathRule.BackendHTTPSettings = nil
	}
}
