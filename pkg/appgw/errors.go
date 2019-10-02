// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import "errors"

var (
	// ErrEmptyConfig is an error.
	ErrEmptyConfig = errors.New("empty App Gateway config (APPG001)")

	// ErrMultipleServiceBackendPortBinding is an error.
	ErrMultipleServiceBackendPortBinding = errors.New("more than one service-backend port binding is not allowed (APPG002)")

	// ErrGeneratingProbes is an error.
	ErrGeneratingProbes = errors.New("unable to generate health probes (APPG003)")

	// ErrGeneratingBackendSettings is an error.
	ErrGeneratingBackendSettings = errors.New("unable to generate backend http settings (APPG004)")

	// ErrGeneratingListeners is an error.
	ErrGeneratingListeners = errors.New("unable to generate frontend listeners (APPG005)")

	// ErrGeneratingRoutingRules is an error.
	ErrGeneratingRoutingRules = errors.New("unable to generate request routing rules (APPG006)")

	// ErrKeyNoDefaults is an error.
	ErrKeyNoDefaults = errors.New("either a DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) must be configured (APPG007)")

	// ErrKeyEitherDefaults is an error.
	ErrKeyEitherDefaults = errors.New("URL Path Map must have either DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) but not both (APPG008)")

	// ErrKeyNoBorR is an error.
	ErrKeyNoBorR = errors.New("A valid path rule must have one of RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings) (APPG009)")

	// ErrKeyEitherBorR is an error.
	ErrKeyEitherBorR = errors.New("A Path Rule must have either RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings) but not both (APPG010)")

	// ErrKeyNoPrivateIP is an error.
	ErrKeyNoPrivateIP = errors.New("A Private IP must be present in the Application Gateway FrontendIPConfiguration if the controller is configured to UsePrivateIP for routing rules (APPG011)")

	// ErrKeyNoPublicIP is an error.
	ErrKeyNoPublicIP = errors.New("A Public IP must be present in the Application Gateway FrontendIPConfiguration (APPG012)")

	// ErrIstioMultipleServiceBackendPortBinding is an error.
	ErrIstioMultipleServiceBackendPortBinding = errors.New("more than one service-backend port binding is not allowed (APPG013)")

	// ErrIstioResolvePortsForServices is an error.
	ErrIstioResolvePortsForServices = errors.New("unable to resolve backend port for some services (APPG014)")

	// ErrCreatingBackendPools is an error.
	ErrCreatingBackendPools = errors.New("unable to generate backend address pools (APPG015)")
)
