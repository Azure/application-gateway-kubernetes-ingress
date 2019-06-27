package fixtures

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func GeURLPathMap() *network.ApplicationGatewayURLPathMap {
	return &network.ApplicationGatewayURLPathMap{
		Name: to.StringPtr("URLPathMap-1"),
		ApplicationGatewayURLPathMapPropertiesFormat: &network.ApplicationGatewayURLPathMapPropertiesFormat{
			// DefaultBackendAddressPool - Default backend address pool resource of URL path map.
			DefaultBackendAddressPool: &network.SubResource{
				ID: to.StringPtr(""),
			},

			// DefaultBackendHTTPSettings - Default backend http settings resource of URL path map.
			DefaultBackendHTTPSettings: &network.SubResource{
				ID: to.StringPtr(""),
			},

			// DefaultRewriteRuleSet - Default Rewrite rule set resource of URL path map.
			DefaultRewriteRuleSet: &network.SubResource{
				ID: to.StringPtr(""),
			},

			// DefaultRedirectConfiguration - Default redirect configuration resource of URL path map.
			DefaultRedirectConfiguration: &network.SubResource{
				ID: to.StringPtr(""),
			},

			// PathRules - Path rule of URL path map resource.
			PathRules: &[]network.ApplicationGatewayPathRule{
				*GetPathRulePathBased(),
				*GetPathRuleBasic(),
			},
		},
	}
}

func GetPathRulePathBased() *network.ApplicationGatewayPathRule {
	return &network.ApplicationGatewayPathRule{
		Name: to.StringPtr("PathRule-1"),
		ApplicationGatewayPathRulePropertiesFormat: &network.ApplicationGatewayPathRulePropertiesFormat{
			// Paths - Path rules of URL path map.
			Paths: &[]string{
				"/foo",
				"/bar",
			},

			// BackendAddressPool - Backend address pool resource of URL path map path rule.
			BackendAddressPool: &network.SubResource{
				ID: to.StringPtr("x/y/z/BackendAddressPool-1"),
			},

			// BackendHTTPSettings - Backend http settings resource of URL path map path rule.
			BackendHTTPSettings: &network.SubResource{
				ID: to.StringPtr("x/y/z/BackendHTTPSettings-1"),
			},

			// RedirectConfiguration - Redirect configuration resource of URL path map path rule.
			RedirectConfiguration: &network.SubResource{
				ID: to.StringPtr("x/y/z/RedirectConfiguration-1"),
			},

			// RewriteRuleSet - Rewrite rule set resource of URL path map path rule.
			RewriteRuleSet: &network.SubResource{
				ID: to.StringPtr("x/y/z/RewriteRuleSet-1"),
			},
		},
	}
}

func GetPathRuleBasic() *network.ApplicationGatewayPathRule {
	return &network.ApplicationGatewayPathRule{
		Name: to.StringPtr("PathRule-1"),
		ApplicationGatewayPathRulePropertiesFormat: &network.ApplicationGatewayPathRulePropertiesFormat{
			// Paths - Path rules of URL path map.
			Paths: nil,

			// BackendAddressPool - Backend address pool resource of URL path map path rule.
			BackendAddressPool: &network.SubResource{
				ID: to.StringPtr("x/y/z/BackendAddressPool-2"),
			},

			// BackendHTTPSettings - Backend http settings resource of URL path map path rule.
			BackendHTTPSettings: &network.SubResource{
				ID: to.StringPtr("x/y/z/BackendHTTPSettings-1"),
			},

			// RedirectConfiguration - Redirect configuration resource of URL path map path rule.
			RedirectConfiguration: &network.SubResource{
				ID: to.StringPtr("x/y/z/RedirectConfiguration-1"),
			},

			// RewriteRuleSet - Rewrite rule set resource of URL path map path rule.
			RewriteRuleSet: &network.SubResource{
				ID: to.StringPtr("x/y/z/RewriteRuleSet-1"),
			},
		},
	}
}
