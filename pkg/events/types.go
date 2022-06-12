// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package events

const (
	// ReasonBackendPortTargetMatch is a reason for an event to be emitted.
	ReasonBackendPortTargetMatch = "BackendPortTargetMatch"

	// ReasonEndpointsEmpty is a reason for an event to be emitted.
	ReasonEndpointsEmpty = "EndpointsEmpty"

	// ReasonIngressServiceTargetMatch is a reason for an event to be emitted.
	ReasonIngressServiceTargetMatch = "IngressServiceTargetMatch"

	// ReasonSecretNotFound is a reason for an event to be emitted.
	ReasonSecretNotFound = "SecretNotFound"

	// ReasonServiceNotFound is a reason for an event to be emitted.
	ReasonServiceNotFound = "ServiceNotFound"

	// ReasonPortResolutionError is a reason for an event to be emitted.
	ReasonPortResolutionError = "PortResolutionError"

	// ReasonNoPrivateIPError is a reason for an event to be emitted.
	ReasonNoPrivateIPError = "NoPrivateIP"

	// ReasonNoPreInstalledSslCertificate is a reason for an event to be emitted.
	ReasonNoPreInstalledSslCertificate = "NoPreInstalledSslCertificate"

	// ReasonNoPreInstalledSslProfile is a reason for an event to be emitted.
	ReasonNoPreInstalledSslProfile = "NoPreInstalledSslProfile"

	// ReasonNoPreInstalledRootCertificate is a reason for an event to be emitted.
	ReasonNoPreInstalledRootCertificate = "NoPreInstalledRootCertificate"

	// ReasonRedirectWithNoTLS is a reason for an event to be emitted.
	ReasonRedirectWithNoTLS = "RedirectWithNoTLS"

	// ReasonUnableToUpdateIngressStatus is a reason for an event to be emitted.
	ReasonUnableToUpdateIngressStatus = "UnableToUpdateIngressStatus"

	// ReasonResetIngressStatus is a reason for an event to be emitted.
	ReasonResetIngressStatus = "ResetIngressStatus"

	// ReasonUnableToResetIngressStatus is a reason for an event to be emitted.
	ReasonUnableToResetIngressStatus = "UnableToResetIngressStatus"

	// ReasonInvalidAnnotation is a reason for an event to be emitted.
	ReasonInvalidAnnotation = "InvalidAnnotation"

	// ReasonUnableToFetchAppGw is a reason for an event to be emitted.
	ReasonUnableToFetchAppGw = "UnableToFetchAppGw"

	// ReasonNoValidIngress is a reason for an event to be emitted.
	ReasonNoValidIngress = "NoValidIngress"

	// ReasonInvalidAppGwConfig is a reason for an event to be emitted.
	ReasonInvalidAppGwConfig = "InvalidAppGwConfig"

	// ReasonFailedApplyingAppGwConfig is a reason for an event to be emitted.
	ReasonFailedApplyingAppGwConfig = "FailedApplyingAppGwConfig"

	// ReasonFailedDeployingAppGw is a reason for an event to be emitted.
	ReasonFailedDeployingAppGw = "FailedDeployingAppGw"

	// ReasonValidatonError is a reason for an event to be emitted.
	ReasonValidatonError = "FailedValidatonError"

	// ReasonARMAuthFailure is a reason for an event to be emitted.
	ReasonARMAuthFailure = "ARMAuthFailure"

	// UnsupportedAppGatewaySKUTier is a reason for an event to be emitted.
	UnsupportedAppGatewaySKUTier = "UnsupportedAppGatewaySKUTier"
)
