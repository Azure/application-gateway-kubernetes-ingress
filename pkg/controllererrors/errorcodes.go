// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controllererrors

// ErrorCodes for different errors in the controller
const (

	// appgw package
	ErrorMultipleServiceBackendPortBinding      ErrorCode = "ErrorMultipleServiceBackendPortBinding"
	ErrorGeneratingProbes                       ErrorCode = "ErrorGeneratingProbes"
	ErrorGeneratingBackendSettings              ErrorCode = "ErrorGeneratingBackendSettings"
	ErrorCreatingBackendPools                   ErrorCode = "ErrorCreatingBackendPools"
	ErrorGeneratingListeners                    ErrorCode = "ErrorGeneratingListeners"
	ErrorGeneratingRoutingRules                 ErrorCode = "ErrorGeneratingRoutingRules"
	ErrorNoDefaults                             ErrorCode = "ErrorNoDefaults"
	ErrorEitherDefaults                         ErrorCode = "ErrorEitherDefaults"
	ErrorNoBackendorRedirect                    ErrorCode = "ErrorNoBackendorRedirect"
	ErrorEitherBackendorRedirect                ErrorCode = "ErrorEitherBackendorRedirect"
	ErrorNoPublicIP                             ErrorCode = "ErrorNoPublicIP"
	ErrorNoPrivateIP                            ErrorCode = "ErrorNoPrivateIP"
	ErrorEmptyConfig                            ErrorCode = "ErrorEmptyConfig"
	ErrorIstioResolvePortsForServices           ErrorCode = "ErrorIstioResolvePortsForServices"
	ErrorIstioMultipleServiceBackendPortBinding ErrorCode = "ErrorIstioMultipleServiceBackendPortBinding"

	// k8sContext package
	ErrorEnpdointsNotFound              ErrorCode = "ErrorEnpdointsNotFound"
	ErrorFetchingEnpdoints              ErrorCode = "ErrorFetchingEnpdoints"
	ErrorInformersNotInitialized        ErrorCode = "ErrorInformersNotInitialized"
	ErrorFailedInitialCacheSync         ErrorCode = "ErrorFailedInitialCacheSync"
	ErrorUpdatingIngressStatus          ErrorCode = "ErrorUpdatingIngressStatus"
	ErrorFetchingNodes                  ErrorCode = "ErrorFetchingNodes"
	ErrorNoNodesFound                   ErrorCode = "ErrorNoNodesFound"
	ErrorUnrecognizedNodeProviderPrefix ErrorCode = "ErrorUnrecognizedNodeProviderPrefix"
	ErrorUnknownSecretType              ErrorCode = "ErrorUnknownSecretType"
	ErrorMalformedSecret                ErrorCode = "ErrorMalformedSecret"
	ErrorCreatingFile                   ErrorCode = "ErrorCreatingFile"
	ErrorWritingToFile                  ErrorCode = "ErrorWritingToFile"
	ErrorExportingWithOpenSSL           ErrorCode = "ErrorExportingWithOpenSSL"
)
