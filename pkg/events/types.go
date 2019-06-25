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
)
