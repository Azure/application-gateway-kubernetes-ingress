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

	// ReasonRedirectWithNoTLS is a reason for an event to be emitted.
	ReasonRedirectWithNoTLS = "RedirectWithNoTLS"

	// ReasonUnableToUpdateIngressStatus is a reason for an event to be emitted.
	ReasonUnableToUpdateIngressStatus = "UnableToUpdateIngressStatus"

	// ReasonInvalidAnnotation is a reason for an event to be emitted.
	ReasonInvalidAnnotation = "InvalidAnnotation"
)
