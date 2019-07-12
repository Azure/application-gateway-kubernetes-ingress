// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	// DefaultRequestRoutingRuleName is a string constant.
	DefaultRequestRoutingRuleName = "rr-80"
)

// GetRequestRoutingRulePathBased creates a new struct for use in unit tests.
func GetRequestRoutingRulePathBased() *n.ApplicationGatewayRequestRoutingRule {
	return &n.ApplicationGatewayRequestRoutingRule{
		Name: to.StringPtr("RequestRoutingRule-1"),
		ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
			// RuleType - Rule type. Possible values include: 'Basic', 'PathBasedRouting'
			RuleType: n.PathBasedRouting,

			// BackendAddressPool - Backend address pool resource of the application gateway.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendAddressPoolName1),
			},

			// BackendHTTPSettings - Backend http settings resource of the application gateway.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/BackendHTTPSettings-1"),
			},

			// HTTPListener - Http listener resource of the application gateway.
			HTTPListener: &n.SubResource{
				ID: to.StringPtr("x/y/z/HTTPListener-PathBased"),
			},

			// URLPathMap - URL path map resource of the application gateway.
			URLPathMap: &n.SubResource{
				ID: to.StringPtr("x/y/z/URLPathMap-1"),
			},

			// RewriteRuleSet - Rewrite Rule Set resource in Basic rule of the application gateway.
			RewriteRuleSet: &n.SubResource{
				ID: to.StringPtr("x/y/z/RewriteRuleSet-1"),
			},

			// RedirectConfiguration - Redirect configuration resource of the application gateway.
			RedirectConfiguration: &n.SubResource{
				ID: to.StringPtr("x/y/z/RedirectConfiguration-1"),
			},
		},
	}
}

// GetRequestRoutingRuleBasic creates a new struct for use in unit tests.
func GetRequestRoutingRuleBasic() *n.ApplicationGatewayRequestRoutingRule {
	return &n.ApplicationGatewayRequestRoutingRule{
		Name: to.StringPtr("RequestRoutingRule-2"),
		ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
			// RuleType - Rule type. Possible values include: 'Basic', 'PathBasedRouting'
			RuleType: n.Basic,

			// BackendAddressPool - Backend address pool resource of the application gateway.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendAddressPoolName2),
			},

			// BackendHTTPSettings - Backend http settings resource of the application gateway.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/BackendHTTPSettings-2"),
			},

			// HTTPListener - Http listener resource of the application gateway.
			HTTPListener: &n.SubResource{
				ID: to.StringPtr("x/y/z/HTTPListener-Basic"),
			},

			// URLPathMap - URL path map resource of the application gateway.
			URLPathMap: nil,

			// RewriteRuleSet - Rewrite Rule Set resource in Basic rule of the application gateway.
			RewriteRuleSet: &n.SubResource{
				ID: to.StringPtr("x/y/z/RewriteRuleSet-2"),
			},

			// RedirectConfiguration - Redirect configuration resource of the application gateway.
			RedirectConfiguration: &n.SubResource{
				ID: to.StringPtr("x/y/z/RedirectConfiguration-2"),
			},
		},
	}
}

// GetDefaultRoutingRule returns the default routing rule.
func GetDefaultRoutingRule() *n.ApplicationGatewayRequestRoutingRule {
	return &n.ApplicationGatewayRequestRoutingRule{
		Name: to.StringPtr(DefaultRequestRoutingRuleName),
		ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
			// RuleType - Rule type. Possible values include: 'Basic', 'PathBasedRouting'
			RuleType: n.Basic,

			// BackendAddressPool - Backend address pool resource of the application gateway.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + DefaultBackendPoolName),
			},

			// BackendHTTPSettings - Backend http settings resource of the application gateway.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + DefaultBackendHTTPSettingsName),
			},

			// HTTPListener - Http listener resource of the application gateway.
			HTTPListener: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + DefaultHTTPListenerName),
			},
		},
	}
}
