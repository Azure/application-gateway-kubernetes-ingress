package annotations

import (
	extensions "k8s.io/api/extensions/v1beta1"
)

const (
	// Prefix AppGw Prefix
	Prefix = "appgw.ingress.kubernetes.io"

	// BackendPathPrefix defines the value for Path which should be used as a prefix for all HTTP requests.
	// Null means no path will be prefixed. Default value is null.
	BackendPathPrefix = "appgw.ingress.kubernetes.io/backend-path-prefix"

	// IngressClass defines the key of the annotation which needs to be set in order to specify
	// that this is an ingress resource meant for the application gateway ingress controller.
	IngressClassKey = "kubernetes.io/ingress.class"

	// IngressControllerName defines the value of the `IngressClass` annotation that will tell the ingress controller
	// whether it should act on this ingress resource or not.
	IngressControllerName = "azure/application-gateway"
)

// IngressAnnotations helper
type IngressAnnotations struct {
	annotations map[string]string
}

// FromIngress Annotations
func FromIngress(ing *extensions.Ingress) *IngressAnnotations {
	return &IngressAnnotations{ing.Annotations}
}

// OverrideBackendPath override path
func (ing *IngressAnnotations) BackendPathPrefix() string {
	val, ok := ing.annotations[BackendPathPrefix]
	if !ok {
		return ""
	}

	return val
}

// IngressClass ingress class
func (ing *IngressAnnotations) IngressClass() string {
	val, ok := ing.annotations[IngressClassKey]
	if !ok {
		return ""
	}
	return val
}
