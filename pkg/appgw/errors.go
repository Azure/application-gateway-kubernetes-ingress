// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import "errors"

var (
	// ErrEmptyConfig is an error.
	ErrEmptyConfig                       = errors.New("empty App Gateway config")

	// ErrMultipleServiceBackendPortBinding is an error.
	ErrMultipleServiceBackendPortBinding = errors.New("more than one service-backend port binding is not allowed")

	// ErrGeneratingProbes is an error.
	ErrGeneratingProbes                  = errors.New("unable to generate health probes")

	// ErrGeneratingBackendSettings is an error.
	ErrGeneratingBackendSettings         = errors.New("unable to generate backend http settings")

	// ErrGeneratingPools is an error.
	ErrGeneratingPools                   = errors.New("unable to generate backend address pools")

	// ErrGeneratingListeners is an error.
	ErrGeneratingListeners               = errors.New("unable to generate frontend listeners")

	// ErrGeneratingRoutingRules is an error.
	ErrGeneratingRoutingRules            = errors.New("unable to generate request routing rules")

	// ErrKeyNoDefaults is an error.
	ErrKeyNoDefaults                     = errors.New("either a DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) must be configured")

	// ErrKeyEitherDefaults is an error.
	ErrKeyEitherDefaults                 = errors.New("URL Path Map must have either DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) but not both")

	// ErrKeyNoBorR is an error.
	ErrKeyNoBorR                         = errors.New("A valid path rule must have one of RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings)")

	// ErrKeyEitherBorR is an error.
	ErrKeyEitherBorR                     = errors.New("A Path Rule must have either RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings) but not both")

	// ErrKeyNoPrivateIP is an error.
	ErrKeyNoPrivateIP                    = errors.New("A Private IP must be present in the Application Gateway FrontendIPConfiguration if the controller is configured to UsePrivateIP for routing rules")

	// ErrKeyNoPublicIP is an error.
	ErrKeyNoPublicIP                     = errors.New("A Public IP must be present in the Application Gateway FrontendIPConfiguration")
)
