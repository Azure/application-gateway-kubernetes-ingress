// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func (c *appGwConfigBuilder) getIstioPathMaps(cbCtx *ConfigBuilderContext) map[listenerIdentifier]*n.ApplicationGatewayURLPathMap {
	defaultAddressPoolID := to.StringPtr(c.appGwIdentifier.addressPoolID(defaultBackendAddressPoolName))
	defaultHTTPSettingsID := to.StringPtr(c.appGwIdentifier.httpSettingsID(defaultBackendHTTPSettingsName))

	// TODO(delqn)
	istioHTTPSettings, _, _, _ := c.getIstioDestinationsAndSettingsMap(cbCtx)

	backendByDestination := c.newIstioBackendPoolMap(cbCtx)

	urlPathMaps := make(map[listenerIdentifier]*n.ApplicationGatewayURLPathMap)
	for virtSvcIdx, virtSvc := range cbCtx.IstioVirtualServices {
		for _, http := range virtSvc.Spec.HTTP {
			// TODO(delqn): consider weights
			for matchIdx, match := range http.Match {
				dst := istioDestinationIdentifier{
					serviceIdentifier: serviceIdentifier{
						Namespace: virtSvc.Namespace,
						Name:      virtSvc.Name,
					},
					istioVirtualServiceIdentifier: istioVirtualServiceIdentifier{
						Namespace: virtSvc.Namespace,
						Name:      virtSvc.Name,
					},
					// TODO(delqn)
					DestinationHost: "httpbin",
					DestinationPort: 8000,
				}

				// TODO(delqn)
				listenerID := listenerIdentifier{
					FrontendPort: 80,
					HostName:     virtSvc.Spec.Hosts[0], // TODO(delqn),
				}
				pool, found := backendByDestination[dst]

				if !found {
					continue
				}
				pathMap := n.ApplicationGatewayURLPathMap{
					Etag: to.StringPtr("*"),
					Name: to.StringPtr(generateURLPathMapName(listenerID)),
					ID:   to.StringPtr(c.appGwIdentifier.urlPathMapID(generateURLPathMapName(listenerID))),
					ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
						DefaultBackendAddressPool:  &n.SubResource{ID: defaultAddressPoolID},
						DefaultBackendHTTPSettings: &n.SubResource{ID: defaultHTTPSettingsID},
						PathRules:                  &[]n.ApplicationGatewayPathRule{},
					},
				}

				pathRuleIdx := fmt.Sprintf("%d-%d", virtSvcIdx, matchIdx)

				pathRule := n.ApplicationGatewayPathRule{
					Etag: to.StringPtr("*"),
					Name: to.StringPtr(generatePathRuleName(virtSvc.Namespace, virtSvc.Name, pathRuleIdx)),
					ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
						Paths: &[]string{
							match.URI.Prefix,
						},
						BackendAddressPool: &n.SubResource{ID: pool.ID},
						// TODO(delqn)
						BackendHTTPSettings: &n.SubResource{ID: istioHTTPSettings[0].ID},
					},
				}
				pathMap.PathRules = &[]n.ApplicationGatewayPathRule{
					pathRule,
				}
				urlPathMaps[listenerID] = &pathMap
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
