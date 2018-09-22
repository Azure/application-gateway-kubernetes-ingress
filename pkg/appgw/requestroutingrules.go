// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"strconv"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) pathMaps(ingress *v1beta1.Ingress, rule *v1beta1.IngressRule,
	frontendListenerID frontendListenerIdentifier, urlPathMap *network.ApplicationGatewayURLPathMap,
	defaultAddressPoolID string, defaultHTTPSettingsID string) *network.ApplicationGatewayURLPathMap {
	if urlPathMap == nil {
		urlPathMapName := generateURLPathMapName(frontendListenerID)
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

	for _, path := range rule.HTTP.Paths {
		backendID := generateBackendID(ingress, &path.Backend)
		backendPool := builder.backendPoolMap[backendID]
		backendHTTPSettings := builder.backendHTTPSettingsMap[backendID]
		if backendPool == nil || backendHTTPSettings == nil {
			continue
		}
		pathRules := *urlPathMap.PathRules

		backendPoolSubResource := network.SubResource{ID: to.StringPtr(builder.appGwIdentifier.addressPoolID(*backendPool.Name))}
		backendHTTPSettingsSubResource := network.SubResource{ID: to.StringPtr(builder.appGwIdentifier.httpSettingsID(*backendHTTPSettings.Name))}

		if len(path.Path) == 0 || path.Path == "/*" {
			// this backend should be a default backend, catches all traffic
			// check if it is a host-specific default backend
			if rule.Host == frontendListenerID.HostName {
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

func (builder *appGwConfigBuilder) RequestRoutingRules(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error) {
	urlPathMaps := make(map[frontendListenerIdentifier]*network.ApplicationGatewayURLPathMap)
	for _, ingress := range ingressList {
		defaultAddressPoolID := builder.appGwIdentifier.addressPoolID(defaultBackendAddressPoolName)
		defaultHTTPSettingsID := builder.appGwIdentifier.httpSettingsID(defaultBackendHTTPSettingsName)

		var wildcardRule *v1beta1.IngressRule
		wildcardRule = nil
		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP != nil && len(rule.Host) == 0 {
				wildcardRule = &rule
			}
		}

		// find the default backend for a ingress
		defBackend := ingress.Spec.Backend
		if wildcardRule != nil {
			// wildcard rule override the default backend
			for _, path := range wildcardRule.HTTP.Paths {
				if path.Path == "" || path.Path == "/*" {
					// look for default path
					defBackend = &path.Backend
				}
			}
		}

		if defBackend != nil {
			// has default backend
			defaultBackendID := generateBackendID(ingress, defBackend)

			defaultHTTPSettings := builder.backendHTTPSettingsMap[defaultBackendID]
			defaultAddressPool := builder.backendPoolMap[defaultBackendID]
			if defaultAddressPool != nil && defaultHTTPSettings != nil {
				// default settings is valid
				defaultAddressPoolID = builder.appGwIdentifier.addressPoolID(*defaultAddressPool.Name)
				defaultHTTPSettingsID = builder.appGwIdentifier.httpSettingsID(*defaultHTTPSettings.Name)
			}
		}

		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				// skip no http rule
				continue
			}

			httpAvailable := false
			httpsAvailable := false

			listenerHTTPID := generateFrontendListenerID(&rule, network.HTTP, nil)
			_, exist := builder.httpListenersMap[listenerHTTPID]
			if exist {
				httpAvailable = true
			}

			// check annotation for port override
			listenerHTTPSID := generateFrontendListenerID(&rule, network.HTTPS, nil)
			_, exist = builder.httpListenersMap[listenerHTTPSID]
			if exist {
				httpsAvailable = true
			}

			// TODO check annotations for disabling http
			if httpAvailable {
				if wildcardRule != nil && len(rule.Host) != 0 {
					// only add wildcard rules when host is specified
					urlPathMaps[listenerHTTPID] = builder.pathMaps(ingress, wildcardRule,
						listenerHTTPID, urlPathMaps[listenerHTTPID],
						defaultAddressPoolID, defaultHTTPSettingsID)
				}
				// need to eliminate non-unique paths
				urlPathMaps[listenerHTTPID] = builder.pathMaps(ingress, &rule,
					listenerHTTPID, urlPathMaps[listenerHTTPID],
					defaultAddressPoolID, defaultHTTPSettingsID)
			}

			if httpsAvailable {
				if wildcardRule != nil && len(rule.Host) != 0 {
					// only add wildcard rules when host is specified
					urlPathMaps[listenerHTTPSID] = builder.pathMaps(ingress, wildcardRule,
						listenerHTTPSID, urlPathMaps[listenerHTTPSID],
						defaultAddressPoolID, defaultHTTPSettingsID)
				}
				// need to eliminate non-unique paths
				urlPathMaps[listenerHTTPSID] = builder.pathMaps(ingress, &rule,
					listenerHTTPSID, urlPathMaps[listenerHTTPSID],
					defaultAddressPoolID, defaultHTTPSettingsID)
			}
		}
	}

	if len(urlPathMaps) == 0 {
		defaultAddressPoolID := builder.appGwIdentifier.addressPoolID(defaultBackendAddressPoolName)
		defaultHTTPSettingsID := builder.appGwIdentifier.httpSettingsID(defaultBackendHTTPSettingsName)
		frontendListenerID := defaultFrontendListenerIdentifier()
		urlPathMapName := generateURLPathMapName(frontendListenerID)
		urlPathMaps[frontendListenerID] = &network.ApplicationGatewayURLPathMap{
			Etag: to.StringPtr("*"),
			Name: &urlPathMapName,
			ApplicationGatewayURLPathMapPropertiesFormat: &network.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool:  &network.SubResource{ID: &defaultAddressPoolID},
				DefaultBackendHTTPSettings: &network.SubResource{ID: &defaultHTTPSettingsID},
				PathRules:                  &[]network.ApplicationGatewayPathRule{},
			},
		}
	}

	urlPathMapFiltered := []network.ApplicationGatewayURLPathMap{}
	requestRoutingRules := []network.ApplicationGatewayRequestRoutingRule{}
	for frontendListenerID, urlPathMap := range urlPathMaps {
		requestRoutingRuleName := generateRequestRoutingRuleName(frontendListenerID)
		httpListener := builder.httpListenersMap[frontendListenerID]
		httpListenerSubResource := network.SubResource{ID: to.StringPtr(builder.appGwIdentifier.httpListenerID(*httpListener.Name))}
		var rule network.ApplicationGatewayRequestRoutingRule
		if len(*urlPathMap.PathRules) == 0 {
			// Basic Rule, because we have no path-based rule
			rule = network.ApplicationGatewayRequestRoutingRule{
				Etag: to.StringPtr("*"),
				Name: &requestRoutingRuleName,
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &network.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType:            network.Basic,
					HTTPListener:        &httpListenerSubResource,
					BackendAddressPool:  urlPathMap.DefaultBackendAddressPool,
					BackendHTTPSettings: urlPathMap.DefaultBackendHTTPSettings,
				},
			}
		} else {
			// Path-based Rule
			urlPathMapSubResource := network.SubResource{ID: to.StringPtr(builder.appGwIdentifier.urlPathMapID(*urlPathMap.Name))}
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

	builder.appGwConfig.RequestRoutingRules = &requestRoutingRules
	builder.appGwConfig.URLPathMaps = &urlPathMapFiltered
	return builder, nil
}
