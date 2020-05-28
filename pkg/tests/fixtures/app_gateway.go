// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// GetAppGateway creates an ApplicationGateway struct.
func GetAppGateway() n.ApplicationGateway {
	// The order of the lists below is important as we reference these by index in unit tests.
	return n.ApplicationGateway{
		ID: to.StringPtr("something"),
		ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{

			RequestRoutingRules: &[]n.ApplicationGatewayRequestRoutingRule{
				*GetDefaultRoutingRule(),
				*GetRequestRoutingRuleBasic(),
				*GetRequestRoutingRulePathBased1(),
				*GetRequestRoutingRulePathBased2(),
				*GetRequestRoutingRulePathBased3(),
			},
			URLPathMaps: &[]n.ApplicationGatewayURLPathMap{
				*GetDefaultURLPathMap(),
				*GetURLPathMap1(),
				*GetURLPathMap2(),
				*GetURLPathMap3(),
			},

			HTTPListeners: &[]n.ApplicationGatewayHTTPListener{
				*GetDefaultListener(),
				*GetListenerBasic(),
				*GetListenerPathBased1(),
				*GetListenerPathBased2(),
				*GetListenerUnassociated(),
				*GetListenerWildcard(),
			},

			SslCertificates: &[]n.ApplicationGatewaySslCertificate{
				GetCertificate1(),
				GetCertificate2(),
				GetCertificate3(),
			},

			TrustedRootCertificates: &[]n.ApplicationGatewayTrustedRootCertificate{
				GetRootCertificate1(),
				GetRootCertificate2(),
				GetRootCertificate3(),
			},

			Probes: &[]n.ApplicationGatewayProbe{
				GetApplicationGatewayProbe(nil, to.StringPtr(PathFoo)), // /foo
				GetApplicationGatewayProbe(nil, to.StringPtr(PathBar)), // /bar
				GetApplicationGatewayProbe(to.StringPtr(tests.OtherHost), nil),
			},

			BackendHTTPSettingsCollection: &[]n.ApplicationGatewayBackendHTTPSettings{
				GetHTTPSettings1(),
				GetHTTPSettings2(),
				GetHTTPSettings3(),
			},

			FrontendIPConfigurations: &[]n.ApplicationGatewayFrontendIPConfiguration{
				GetPublicIPConfiguration(),
			},

			RedirectConfigurations: &[]n.ApplicationGatewayRedirectConfiguration{
				{
					Name: to.StringPtr("redirect-1"),
					ApplicationGatewayRedirectConfigurationPropertiesFormat: &n.ApplicationGatewayRedirectConfigurationPropertiesFormat{},
				},
			},
		},
	}
}
