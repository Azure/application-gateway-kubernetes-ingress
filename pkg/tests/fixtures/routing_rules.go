// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	// DefaultRequestRoutingRuleName is a string constant.
	DefaultRequestRoutingRuleName = "rr-80"

	// RequestRoutingRuleName1 is a string constant.
	RequestRoutingRuleName1 = "RequestRoutingRule-1"

	// RequestRoutingRuleName2 is a string constant.
	RequestRoutingRuleName2 = "RequestRoutingRule-2"

	// RequestRoutingRuleName3 is a string constant.
	RequestRoutingRuleName3 = "RequestRoutingRule-3"
)

// GetRequestRoutingRulePathBased1 creates a new struct for use in unit tests.
func GetRequestRoutingRulePathBased1() *n.ApplicationGatewayRequestRoutingRule {
	return &n.ApplicationGatewayRequestRoutingRule{
		Name: to.StringPtr(RequestRoutingRuleName1),
		ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
			// RuleType - Rule type. Possible values include: 'Basic', 'PathBasedRouting'
			RuleType: n.ApplicationGatewayRequestRoutingRuleTypePathBasedRouting,

			// BackendAddressPool - Backend address pool resource of the application gateway.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendAddressPoolName1),
			},

			// BackendHTTPSettings - Backend http settings resource of the application gateway.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendHTTPSettingsName1),
			},

			// HTTPListener - Http listener resource of the application gateway.
			HTTPListener: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + HTTPListenerPathBased1),
			},

			// URLPathMap - URL path map resource of the application gateway.
			URLPathMap: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + URLPathMapName1),
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

// GetRequestRoutingRulePathBased2 creates a new struct for use in unit tests.
func GetRequestRoutingRulePathBased2() *n.ApplicationGatewayRequestRoutingRule {
	return &n.ApplicationGatewayRequestRoutingRule{
		Name: to.StringPtr(RequestRoutingRuleName2),
		ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
			// RuleType - Rule type. Possible values include: 'Basic', 'PathBasedRouting'
			RuleType: n.ApplicationGatewayRequestRoutingRuleTypePathBasedRouting,

			// BackendAddressPool - Backend address pool resource of the application gateway.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendAddressPoolName1),
			},

			// BackendHTTPSettings - Backend http settings resource of the application gateway.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendHTTPSettingsName1),
			},

			// HTTPListener - Http listener resource of the application gateway.
			HTTPListener: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + HTTPListenerPathBased2),
			},

			// URLPathMap - URL path map resource of the application gateway.
			URLPathMap: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + URLPathMapName2),
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

// GetRequestRoutingRulePathBased3 creates a new struct for use in unit tests.
func GetRequestRoutingRulePathBased3() *n.ApplicationGatewayRequestRoutingRule {
	return &n.ApplicationGatewayRequestRoutingRule{
		Name: to.StringPtr(RequestRoutingRuleName3),
		ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
			// RuleType - Rule type. Possible values include: 'Basic', 'PathBasedRouting'
			RuleType: n.ApplicationGatewayRequestRoutingRuleTypePathBasedRouting,

			// BackendAddressPool - Backend address pool resource of the application gateway.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendAddressPoolName1),
			},

			// BackendHTTPSettings - Backend http settings resource of the application gateway.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendHTTPSettingsName1),
			},

			// HTTPListener - Http listener resource of the application gateway.
			HTTPListener: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + HTTPListenerWildcard),
			},

			// URLPathMap - URL path map resource of the application gateway.
			URLPathMap: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + URLPathMapName3),
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
			RuleType: n.ApplicationGatewayRequestRoutingRuleTypeBasic,

			// BackendAddressPool - Backend address pool resource of the application gateway.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendAddressPoolName2),
			},

			// BackendHTTPSettings - Backend http settings resource of the application gateway.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendHTTPSettingsName2),
			},

			// HTTPListener - Http listener resource of the application gateway.
			HTTPListener: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + HTTPListenerNameBasic),
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
			RuleType: n.ApplicationGatewayRequestRoutingRuleTypeBasic,

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
