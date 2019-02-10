package annotations

import (
	"k8s.io/api/extensions/v1beta1"
)

const (
	// BackendPathPrefixKey defines the value for Path which should be used as a prefix for all HTTP requests.
	// Null means no path will be prefixed. Default value is null.
	BackendPathPrefixKey = "appgw.ingress.kubernetes.io/backend-path-prefix"

	// IngressClassKey defines the key of the annotation which needs to be set in order to specify
	// that this is an ingress resource meant for the application gateway ingress controller.
	IngressClassKey = "kubernetes.io/ingress.class"

	// ApplicationGatewayIngressClass defines the value of the `IngressClassKey` annotation that will tell the ingress controller
	// whether it should act on this ingress resource or not.
	ApplicationGatewayIngressClass = "azure/application-gateway"
)

// BackendPathPrefix override path
func BackendPathPrefix(ing *v1beta1.Ingress) string {
	val, _ := ing.Annotations[BackendPathPrefixKey]
	return val
}

// IngressClass ingress class
func IngressClass(ing *v1beta1.Ingress) string {
	val, _ := ing.Annotations[IngressClassKey]
	return val
}

// IsApplicationGatewayIngress checks if the Ingress resource can be handled by the Application Gateway ingress controller.
func IsApplicationGatewayIngress(ing *v1beta1.Ingress) bool {
	controllerName := ing.Annotations[IngressClassKey]
	return controllerName == ApplicationGatewayIngressClass
}
