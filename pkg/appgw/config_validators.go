// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"errors"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

const (
	errKeyNoDefaults     = "no-defaults"
	errKeyEitherDefaults = "either-defaults"
	errKeyNoBorR         = "no-backend-or-redirect"
	errKeyEitherBorR     = "either-backend-or-redirect"
)

var validationErrors = map[string]error{
	errKeyNoDefaults:     errors.New("either a DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) must be configured"),
	errKeyEitherDefaults: errors.New("URL Path Map must have either DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) but not both"),
	errKeyNoBorR:         errors.New("A valid path rule must have one of RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings)"),
	errKeyEitherBorR:     errors.New("A Path Rule must have either RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings) but not both"),
}

func validateURLPathMaps(config *n.ApplicationGatewayPropertiesFormat) error {
	if config.URLPathMaps == nil {
		return nil
	}

	for _, pathMap := range *config.URLPathMaps {
		if len(*pathMap.PathRules) == 0 {
			// There are no paths. This is a rule of type "Basic"
			validRedirect := pathMap.DefaultRedirectConfiguration != nil
			validBackend := pathMap.DefaultBackendAddressPool != nil && pathMap.DefaultBackendHTTPSettings != nil

			if !validRedirect && !validBackend {
				return validationErrors[errKeyNoDefaults]
			}

			if validRedirect && validBackend || !validRedirect && !validBackend {
				return validationErrors[errKeyEitherDefaults]
			}

		} else {
			// There are paths defined. This is a rule of type "Path-based"
			for _, rule := range *pathMap.PathRules {
				validRedirect := rule.RedirectConfiguration != nil
				validBackend := rule.BackendAddressPool != nil && rule.BackendHTTPSettings != nil

				if !validRedirect && !validBackend {
					return validationErrors[errKeyNoBorR]
				}

				if validRedirect && validBackend || !validRedirect && !validBackend {
					return validationErrors[errKeyEitherBorR]
				}
			}

		}
	}
	return nil
}
