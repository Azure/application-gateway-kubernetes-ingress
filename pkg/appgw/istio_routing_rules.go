// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"strconv"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func (c *appGwConfigBuilder) getIstioPathMaps(cbCtx *ConfigBuilderContext) map[listenerIdentifier]*n.ApplicationGatewayURLPathMap {
	defaultAddressPoolID := to.StringPtr(c.appGwIdentifier.AddressPoolID(DefaultBackendAddressPoolName))
	defaultHTTPSettingsID := to.StringPtr(c.appGwIdentifier.HTTPSettingsID(DefaultBackendHTTPSettingsName))

	// TODO(delqn)
	istioHTTPSettings, _, _, _ := c.getIstioDestinationsAndSettingsMap(cbCtx)

	backendByDestination := c.newIstioBackendPoolMap(cbCtx)

	urlPathMaps := make(map[listenerIdentifier]*n.ApplicationGatewayURLPathMap)
	for virtSvcIdx, virtSvc := range cbCtx.IstioVirtualServices {
		for _, http := range virtSvc.Spec.HTTP {
			// TODO(delqn): consider weights
			host := http.Route[0].Destination.Host
			var port uint32
			if http.Route[0].Destination.Port.Number != 0 {
				port = http.Route[0].Destination.Port.Number
			} else {
				port64, _ := strconv.ParseUint(http.Route[0].Destination.Port.Name, 10, 32)
				port = uint32(port64)
			}
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
					DestinationHost: host,
					DestinationPort: port,
				}

				// TODO(delqn)
				listenerID := listenerIdentifier{
					FrontendPort: 80,
					HostNames:    [5]string{virtSvc.Spec.Hosts[0]}, // TODO(delqn),
				}
				pool, found := backendByDestination[dst]

				if !found {
					continue
				}
				pathMapName := generateURLPathMapName(listenerID)
				pathMap := n.ApplicationGatewayURLPathMap{
					Etag: to.StringPtr("*"),
					Name: to.StringPtr(pathMapName),
					ID:   to.StringPtr(c.appGwIdentifier.urlPathMapID(pathMapName)),
					ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
						DefaultBackendAddressPool:  &n.SubResource{ID: defaultAddressPoolID},
						DefaultBackendHTTPSettings: &n.SubResource{ID: defaultHTTPSettingsID},
						PathRules:                  &[]n.ApplicationGatewayPathRule{},
					},
				}

				pathRuleIdx := fmt.Sprintf("%d-%d", virtSvcIdx, matchIdx)

				pathRuleName := generatePathRuleName(virtSvc.Namespace, virtSvc.Name, pathRuleIdx)
				pathRule := n.ApplicationGatewayPathRule{
					Etag: to.StringPtr("*"),
					Name: to.StringPtr(pathRuleName),
					ID:   to.StringPtr(c.appGwIdentifier.pathRuleID(pathMapName, pathRuleName)),
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
		defaultAddressPoolID := c.appGwIdentifier.AddressPoolID(DefaultBackendAddressPoolName)
		defaultHTTPSettingsID := c.appGwIdentifier.HTTPSettingsID(DefaultBackendHTTPSettingsName)
		listenerID := defaultFrontendListenerIdentifier()
		pathMapName := generateURLPathMapName(listenerID)
		urlPathMaps[listenerID] = &n.ApplicationGatewayURLPathMap{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(pathMapName),
			ID:   to.StringPtr(c.appGwIdentifier.urlPathMapID(pathMapName)),
			ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
				DefaultBackendAddressPool:  &n.SubResource{ID: &defaultAddressPoolID},
				DefaultBackendHTTPSettings: &n.SubResource{ID: &defaultHTTPSettingsID},
				PathRules:                  &[]n.ApplicationGatewayPathRule{},
			},
		}
	}

	return urlPathMaps
}
