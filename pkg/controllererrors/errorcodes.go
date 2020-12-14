// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controllererrors

// ErrorCodes for different errors in the controller
const (

	// appgw package
	ErrorServiceNotFound                           ErrorCode = "ServiceNotFound"
	ErrorMultipleServiceBackendPortBinding         ErrorCode = "ErrorMultipleServiceBackendPortBinding"
	ErrorUnableToResolveBackendPortFromServicePort ErrorCode = "ErrorUnableToResolveBackendPortFromServicePort"
	ErrorGeneratingProbes                          ErrorCode = "ErrorGeneratingProbes"
	ErrorGeneratingBackendSettings                 ErrorCode = "ErrorGeneratingBackendSettings"
	ErrorCreatingBackendPools                      ErrorCode = "ErrorCreatingBackendPools"
	ErrorGeneratingListeners                       ErrorCode = "ErrorGeneratingListeners"
	ErrorGeneratingRoutingRules                    ErrorCode = "ErrorGeneratingRoutingRules"
	ErrorNoDefaults                                ErrorCode = "ErrorNoDefaults"
	ErrorEitherDefaults                            ErrorCode = "ErrorEitherDefaults"
	ErrorNoBackendorRedirect                       ErrorCode = "ErrorNoBackendorRedirect"
	ErrorEitherBackendorRedirect                   ErrorCode = "ErrorEitherBackendorRedirect"
	ErrorNoPublicIP                                ErrorCode = "ErrorNoPublicIP"
	ErrorNoPrivateIP                               ErrorCode = "ErrorNoPrivateIP"
	ErrorEmptyConfig                               ErrorCode = "ErrorEmptyConfig"
	ErrorIstioResolvePortsForServices              ErrorCode = "ErrorIstioResolvePortsForServices"
	ErrorIstioMultipleServiceBackendPortBinding    ErrorCode = "ErrorIstioMultipleServiceBackendPortBinding"

	// k8sContext package
	ErrorEnpdointsNotFound              ErrorCode = "ErrorEnpdointsNotFound"
	ErrorFetchingEnpdoints              ErrorCode = "ErrorFetchingEnpdoints"
	ErrorFetchingBackendAddressPool     ErrorCode = "ErrorFetchingBackendAddressPool"
	ErrorFetchingInstanceUpdateStatus   ErrorCode = "ErrorFetchingInstanceUpdateStatus"
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

	// brownfield package
	ErrorListenerLookup ErrorCode = "ErrorListenerLookup"

	// environment package
	ErrorMissingApplicationGatewayNameOrApplicationGatewayID ErrorCode = "ErrorMissingApplicationGatewayNameOrApplicationGatewayID"
	ErrorMissingApplicationGatewayName                       ErrorCode = "ErrorMissingApplicationGatewayName"
	ErrorNotAllowedApplicationGatewayID                      ErrorCode = "ErrorNotAllowedApplicationGatewayID"
	ErrorMissingSubnetInfo                                   ErrorCode = "ErrorMissingSubnetInfo"
	ErrorInvalidReconcilePeriod                              ErrorCode = "ErrorInvalidReconcilePeriod"

	// controller package
	ErrorFetchingAppGatewayConfig  ErrorCode = "ErrorFetchingAppGatewayConfig"
	ErrorDeployingAppGatewayConfig ErrorCode = "ErrorDeployingAppGatewayConfig"

	// annotations package
	ErrorMissingAnnotation ErrorCode = "ErrorMissingAnnotation"
	ErrorInvalidContent    ErrorCode = "ErrorInvalidContent"

	// azure package
	ErrorGetApplicationGatewayError             ErrorCode = "ErrorGetApplicationGatewayError"
	ErrorApplicationGatewayNotFound             ErrorCode = "ErrorApplicationGatewayNotFound"
	ErrorApplicationGatewayForbidden            ErrorCode = "ErrorApplicationGatewayForbidden"
	ErrorApplicationGatewayUnexpectedStatusCode ErrorCode = "ErrorApplicationGatewayUnexpectedStatusCode"
	ErrorSubnetNotFound                         ErrorCode = "ErrorSubnetNotFound"
	ErrorMissingResourceGroup                   ErrorCode = "ErrorMissingResourceGroup"

	// main package
	ErrorNoSuchNamespace ErrorCode = "ErrorNoSuchNamespace"
)
