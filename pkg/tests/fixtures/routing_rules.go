package fixtures

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func GetRequestRoutingRulePathBased() *network.ApplicationGatewayRequestRoutingRule {
	return &network.ApplicationGatewayRequestRoutingRule{
		Name: to.StringPtr("RequestRoutingRule-1"),
		ApplicationGatewayRequestRoutingRulePropertiesFormat: &network.ApplicationGatewayRequestRoutingRulePropertiesFormat{
			// RuleType - Rule type. Possible values include: 'Basic', 'PathBasedRouting'
			RuleType: network.PathBasedRouting,

			// BackendAddressPool - Backend address pool resource of the application gateway.
			BackendAddressPool: &network.SubResource{
				ID: to.StringPtr("x/y/z/BackendAddressPool-1"),
			},

			// BackendHTTPSettings - Backend http settings resource of the application gateway.
			BackendHTTPSettings: &network.SubResource{
				ID: to.StringPtr("x/y/z/BackendHTTPSettings-1"),
			},

			// HTTPListener - Http listener resource of the application gateway.
			HTTPListener: &network.SubResource{
				ID: to.StringPtr("x/y/z/HTTPListener-PathBased"),
			},

			// URLPathMap - URL path map resource of the application gateway.
			URLPathMap: &network.SubResource{
				ID: to.StringPtr("x/y/z/URLPathMap-1"),
			},

			// RewriteRuleSet - Rewrite Rule Set resource in Basic rule of the application gateway.
			RewriteRuleSet: &network.SubResource{
				ID: to.StringPtr("x/y/z/RewriteRuleSet-1"),
			},

			// RedirectConfiguration - Redirect configuration resource of the application gateway.
			RedirectConfiguration: &network.SubResource{
				ID: to.StringPtr("x/y/z/RedirectConfiguration-1"),
			},
		},
	}
}

func GetRequestRoutingRuleBasic() *network.ApplicationGatewayRequestRoutingRule {
	return &network.ApplicationGatewayRequestRoutingRule{
		Name: to.StringPtr("RequestRoutingRule-2"),
		ApplicationGatewayRequestRoutingRulePropertiesFormat: &network.ApplicationGatewayRequestRoutingRulePropertiesFormat{
			// RuleType - Rule type. Possible values include: 'Basic', 'PathBasedRouting'
			RuleType: network.Basic,

			// BackendAddressPool - Backend address pool resource of the application gateway.
			BackendAddressPool: &network.SubResource{
				ID: to.StringPtr("x/y/z/BackendAddressPool-2"),
			},

			// BackendHTTPSettings - Backend http settings resource of the application gateway.
			BackendHTTPSettings: &network.SubResource{
				ID: to.StringPtr("x/y/z/BackendHTTPSettings-2"),
			},

			// HTTPListener - Http listener resource of the application gateway.
			HTTPListener: &network.SubResource{
				ID: to.StringPtr("x/y/z/HTTPListener-Basic"),
			},

			// URLPathMap - URL path map resource of the application gateway.
			URLPathMap: nil,

			// RewriteRuleSet - Rewrite Rule Set resource in Basic rule of the application gateway.
			RewriteRuleSet: &network.SubResource{
				ID: to.StringPtr("x/y/z/RewriteRuleSet-2"),
			},

			// RedirectConfiguration - Redirect configuration resource of the application gateway.
			RedirectConfiguration: &network.SubResource{
				ID: to.StringPtr("x/y/z/RedirectConfiguration-2"),
			},
		},
	}
}
