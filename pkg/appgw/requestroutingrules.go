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
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

func (c *appGwConfigBuilder) pathMaps(ingress *v1beta1.Ingress, serviceList []*v1.Service, rule *v1beta1.IngressRule,
	listenerID listenerIdentifier, urlPathMap *network.ApplicationGatewayURLPathMap,
	defaultAddressPoolID string, defaultHTTPSettingsID string) *network.ApplicationGatewayURLPathMap {
	if urlPathMap == nil {
		urlPathMap = &network.ApplicationGatewayURLPathMap{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(generateURLPathMapName(listenerID)),
			ApplicationGatewayURLPathMapPropertiesFormat: &network.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool:  &network.SubResource{ID: &defaultAddressPoolID},
				DefaultBackendHTTPSettings: &network.SubResource{ID: &defaultHTTPSettingsID},
			},
		}
	}

	if urlPathMap.ApplicationGatewayURLPathMapPropertiesFormat.PathRules == nil {
		urlPathMap.PathRules = &[]network.ApplicationGatewayPathRule{}
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
			pathRules = append(pathRules, network.ApplicationGatewayPathRule{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(generatePathRuleName(ingress.Namespace, ingress.Name, strconv.Itoa(pathIdx))),
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

func (c *appGwConfigBuilder) RequestRoutingRules(ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error {
	_, httpListenersMap := c.getListeners(ingressList)
	urlPathMaps := make(map[listenerIdentifier]*network.ApplicationGatewayURLPathMap)
	backendPools := c.newBackendPoolMap(ingressList, serviceList)
	_, backendHTTPSettingsMap, _, _ := c.getBackendsAndSettingsMap(ingressList, serviceList)
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

			listenerHTTPID := generateListenerID(rule, network.HTTP, nil)
			_, httpAvailable := httpListenersMap[listenerHTTPID]

			listenerHTTPSID := generateListenerID(rule, network.HTTPS, nil)
			_, httpsAvailable := httpListenersMap[listenerHTTPSID]

			if httpAvailable {
				if wildcardRule != nil && len(rule.Host) != 0 {
					// only add wildcard rules when host is specified
					urlPathMaps[listenerHTTPID] = c.pathMaps(ingress, serviceList, wildcardRule,
						listenerHTTPID, urlPathMaps[listenerHTTPID],
						defaultAddressPoolID, defaultHTTPSettingsID)
				}

				// need to eliminate non-unique paths
				urlPathMaps[listenerHTTPID] = c.pathMaps(ingress, serviceList, rule,
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
					urlPathMaps[listenerHTTPSID] = c.pathMaps(ingress, serviceList, wildcardRule,
						listenerHTTPSID, urlPathMaps[listenerHTTPSID],
						defaultAddressPoolID, defaultHTTPSettingsID)
				}

				// need to eliminate non-unique paths
				urlPathMaps[listenerHTTPSID] = c.pathMaps(ingress, serviceList, rule,
					listenerHTTPSID, urlPathMaps[listenerHTTPSID],
					defaultAddressPoolID, defaultHTTPSettingsID)
			}
		}
	}

	if len(urlPathMaps) == 0 {
		defaultAddressPoolID := c.appGwIdentifier.addressPoolID(defaultBackendAddressPoolName)
		defaultHTTPSettingsID := c.appGwIdentifier.httpSettingsID(defaultBackendHTTPSettingsName)
		listenerID := defaultFrontendListenerIdentifier()
		urlPathMaps[listenerID] = &network.ApplicationGatewayURLPathMap{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(generateURLPathMapName(listenerID)),
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
		httpListener := httpListenersMap[listenerID]
		if len(*urlPathMap.PathRules) == 0 {
			// Basic Rule, because we have no path-based rule
			rule := network.ApplicationGatewayRequestRoutingRule{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(generateRequestRoutingRuleName(listenerID)),
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &network.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType:              network.Basic,
					HTTPListener:          &network.SubResource{ID: to.StringPtr(c.appGwIdentifier.listenerID(*httpListener.Name))},
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
			rule := network.ApplicationGatewayRequestRoutingRule{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(generateRequestRoutingRuleName(listenerID)),
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &network.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType:     network.PathBasedRouting,
					HTTPListener: &network.SubResource{ID: to.StringPtr(c.appGwIdentifier.listenerID(*httpListener.Name))},
					URLPathMap:   &network.SubResource{ID: to.StringPtr(c.appGwIdentifier.urlPathMapID(*urlPathMap.Name))},
				},
			}
			urlPathMapFiltered = append(urlPathMapFiltered, *urlPathMap)
			requestRoutingRules = append(requestRoutingRules, rule)
		}
	}

	sort.Sort(sorter.ByRequestRoutingRuleName(requestRoutingRules))
	c.appGwConfig.RequestRoutingRules = &requestRoutingRules

	sort.Sort(sorter.ByPathMap(urlPathMapFiltered))
	c.appGwConfig.URLPathMaps = &urlPathMapFiltered

	return nil
}

func (c *appGwConfigBuilder) getSslRedirectConfigResourceReference(targetListener listenerIdentifier) *network.SubResource {
	configName := generateSSLRedirectConfigurationName(targetListener)
	sslRedirectConfigID := c.appGwIdentifier.redirectConfigurationID(configName)
	return resourceRef(sslRedirectConfigID)
}

func (c *appGwConfigBuilder) modifyPathRulesForRedirection(httpURLPathMap *network.ApplicationGatewayURLPathMap, targetListener listenerIdentifier) {
	// Application Gateway supports Basic and Path-based rules

	if len(*httpURLPathMap.PathRules) == 0 {
		// There are no paths. This is a rule of type "Basic"
		redirectRef := c.getSslRedirectConfigResourceReference(targetListener)
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
		redirectRef := c.getSslRedirectConfigResourceReference(targetListener)
		glog.Infof("Attaching redirection config %s request routing rule: %s\n", *redirectRef.ID, *pathRule.Name)

		// A Path Rule must have either RedirectConfiguration xor (BackendAddressPool + BackendHTTPSettings)
		pathRule.RedirectConfiguration = redirectRef

		// Since this is a redirect - ensure Backend is NOT setup
		pathRule.BackendAddressPool = nil
		pathRule.BackendHTTPSettings = nil
	}
}
