// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	// DefaultPathMapName is a string constant.
	DefaultPathMapName = "default-pathmap-name"

	// URLPathMapName1 is a string constant.
	URLPathMapName1 = "URLPathMap-1"

	// URLPathMapName2 is a string constant.
	URLPathMapName2 = "URLPathMap-2"

	// PathRuleName is a string constant.
	PathRuleName = "PathRule-1"

	// PathRuleNameBasic is a string constant.
	PathRuleNameBasic = "PathRule-Basic"
)

// GetURLPathMap1 creates a new struct for use in unit tests.
func GetURLPathMap1() *n.ApplicationGatewayURLPathMap {
	return &n.ApplicationGatewayURLPathMap{
		Name: to.StringPtr(URLPathMapName1),
		ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
			// DefaultBackendAddressPool - Default backend address pool resource of URL path map.
			DefaultBackendAddressPool: &n.SubResource{
				ID: to.StringPtr(""),
			},

			// DefaultBackendHTTPSettings - Default backend http settings resource of URL path map.
			DefaultBackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr(BackendHTTPSettingsName1),
			},

			// DefaultRewriteRuleSet - Default Rewrite rule set resource of URL path map.
			DefaultRewriteRuleSet: &n.SubResource{
				ID: to.StringPtr(""),
			},

			// DefaultRedirectConfiguration - Default redirect configuration resource of URL path map.
			DefaultRedirectConfiguration: &n.SubResource{
				ID: to.StringPtr(""),
			},

			// PathRules - Path rule of URL path map resource.
			PathRules: &[]n.ApplicationGatewayPathRule{
				*GetPathRulePathBased1(),
				*GetPathRuleBasic(),
			},
		},
	}
}

// GetPathRulePathBased1 creates a new struct for use in unit tests.
func GetPathRulePathBased1() *n.ApplicationGatewayPathRule {
	return &n.ApplicationGatewayPathRule{
		Name: to.StringPtr(PathRuleName),
		ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
			// Paths - Path rules of URL path map.
			Paths: &[]string{
				PathFoo,
				PathBar,
				PathBaz,
			},

			// BackendAddressPool - Backend address pool resource of URL path map path rule.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendAddressPoolName1),
			},

			// BackendHTTPSettings - Backend http settings resource of URL path map path rule.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/BackendHTTPSettings-1"),
			},

			// RedirectConfiguration - Redirect configuration resource of URL path map path rule.
			RedirectConfiguration: &n.SubResource{
				ID: to.StringPtr("x/y/z/RedirectConfiguration-1"),
			},

			// RewriteRuleSet - Rewrite rule set resource of URL path map path rule.
			RewriteRuleSet: &n.SubResource{
				ID: to.StringPtr("x/y/z/RewriteRuleSet-1"),
			},
		},
	}
}

// GetPathRuleBasic creates a new struct for use in unit tests.
func GetPathRuleBasic() *n.ApplicationGatewayPathRule {
	return &n.ApplicationGatewayPathRule{
		Name: to.StringPtr(PathRuleNameBasic),
		ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
			// Paths - Path rules of URL path map.
			Paths: nil,

			// BackendAddressPool - Backend address pool resource of URL path map path rule.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendAddressPoolName2),
			},

			// BackendHTTPSettings - Backend http settings resource of URL path map path rule.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/BackendHTTPSettings-1"),
			},

			// RedirectConfiguration - Redirect configuration resource of URL path map path rule.
			RedirectConfiguration: &n.SubResource{
				ID: to.StringPtr("x/y/z/RedirectConfiguration-1"),
			},

			// RewriteRuleSet - Rewrite rule set resource of URL path map path rule.
			RewriteRuleSet: &n.SubResource{
				ID: to.StringPtr("x/y/z/RewriteRuleSet-1"),
			},
		},
	}
}

// GetDefaultURLPathMap makes a default ApplicationGatewayURLPathMap.
func GetDefaultURLPathMap() *n.ApplicationGatewayURLPathMap {
	return &n.ApplicationGatewayURLPathMap{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(DefaultPathMapName),
		ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
			DefaultBackendAddressPool:  &n.SubResource{ID: to.StringPtr("/" + DefaultBackendPoolName)},
			DefaultBackendHTTPSettings: &n.SubResource{ID: to.StringPtr("/" + DefaultBackendHTTPSettingsName)},
		},
	}
}

// GetURLPathMap2 creates a new struct for use in unit tests.
func GetURLPathMap2() *n.ApplicationGatewayURLPathMap {
	return &n.ApplicationGatewayURLPathMap{
		Name: to.StringPtr(URLPathMapName2),
		ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
			// DefaultBackendAddressPool - Default backend address pool resource of URL path map.
			DefaultBackendAddressPool: &n.SubResource{
				ID: to.StringPtr(""),
			},

			// DefaultBackendHTTPSettings - Default backend http settings resource of URL path map.
			DefaultBackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr(BackendHTTPSettingsName2),
			},

			// DefaultRewriteRuleSet - Default Rewrite rule set resource of URL path map.
			DefaultRewriteRuleSet: &n.SubResource{
				ID: to.StringPtr(""),
			},

			// DefaultRedirectConfiguration - Default redirect configuration resource of URL path map.
			DefaultRedirectConfiguration: &n.SubResource{
				ID: to.StringPtr(""),
			},

			// PathRules - Path rule of URL path map resource.
			PathRules: &[]n.ApplicationGatewayPathRule{
				*GetPathRulePathBased2(),
			},
		},
	}
}

// GetPathRulePathBased2 creates a new struct for use in unit tests.
func GetPathRulePathBased2() *n.ApplicationGatewayPathRule {
	return &n.ApplicationGatewayPathRule{
		Name: to.StringPtr(PathRuleName),
		ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
			// Paths - Path rules of URL path map.
			Paths: &[]string{
				PathFox,
			},

			// BackendAddressPool - Backend address pool resource of URL path map path rule.
			BackendAddressPool: &n.SubResource{
				ID: to.StringPtr("x/y/z/" + BackendAddressPoolName1),
			},

			// BackendHTTPSettings - Backend http settings resource of URL path map path rule.
			BackendHTTPSettings: &n.SubResource{
				ID: to.StringPtr("x/y/z/BackendHTTPSettings-1"),
			},

			// RedirectConfiguration - Redirect configuration resource of URL path map path rule.
			RedirectConfiguration: &n.SubResource{
				ID: to.StringPtr("x/y/z/RedirectConfiguration-1"),
			},

			// RewriteRuleSet - Rewrite rule set resource of URL path map path rule.
			RewriteRuleSet: &n.SubResource{
				ID: to.StringPtr("x/y/z/RewriteRuleSet-1"),
			},
		},
	}
}
