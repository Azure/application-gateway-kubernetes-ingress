// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import "errors"

var (
	ErrEmptyConfig                       = errors.New("empty App Gateway config")
	ErrResolvingBackendPortForService    = errors.New("unable to resolve backend port for some services")
	ErrMultipleServiceBackendPortBinding = errors.New("more than one service-backend port binding is not allowed")
	ErrGeneratingProbes                  = errors.New("unable to generate health probes")
	ErrGeneratingBackendSettings         = errors.New("unable to generate backend http settings")
	ErrGeneratingPools                   = errors.New("unable to generate backend address pools")
	ErrGeneratingListeners               = errors.New("unable to generate frontend listeners")
	ErrGeneratingRoutingRules            = errors.New("unable to generate request routing rules")
	ErrKeyNoDefaults                     = errors.New("either a DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) must be configured")
	ErrKeyEitherDefaults                 = errors.New("URL Path Map must have either DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) but not both")
	ErrKeyNoBorR                         = errors.New("A valid path rule must have one of RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings)")
	ErrKeyEitherBorR                     = errors.New("A Path Rule must have either RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings) but not both")
	ErrKeyNoPrivateIP                    = errors.New("A Private IP must be present in the Application Gateway FrontendIPConfiguration if the controller is configured to UsePrivateIP for routing rules")
	ErrKeyNoPublicIP                     = errors.New("A Public IP must be present in the Application Gateway FrontendIPConfiguration")
)
