// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package annotations

import (
	"strconv"

	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/errors"
)

const (
	// ApplicationGatewayPrefix defines the prefix for all keys associated with Application Gateway Ingress controller.
	ApplicationGatewayPrefix = "appgw.ingress.kubernetes.io"

	// BackendPathPrefixKey defines the key for Path which should be used as a prefix for all HTTP requests.
	// Null means no path will be prefixed. Default value is null.
	BackendPathPrefixKey = ApplicationGatewayPrefix + "/backend-path-prefix"

	// SslRedirectKey defines the key for defining with SSL redirect should be turned on for an HTTP endpoint.
	SslRedirectKey = ApplicationGatewayPrefix + "/ssl-redirect"

	// IngressClassKey defines the key of the annotation which needs to be set in order to specify
	// that this is an ingress resource meant for the application gateway ingress controller.
	IngressClassKey = "kubernetes.io/ingress.class"

	// ApplicationGatewayIngressClass defines the value of the `IngressClassKey` annotation that will tell the ingress controller
	// whether it should act on this ingress resource or not.
	ApplicationGatewayIngressClass = "azure/application-gateway"
)

// IngressClass ingress class
func IngressClass(ing *v1beta1.Ingress) (string, error) {
	return parseString(ing, IngressClassKey)
}

// IsApplicationGatewayIngress checks if the Ingress resource can be handled by the Application Gateway ingress controller.
func IsApplicationGatewayIngress(ing *v1beta1.Ingress) (bool, error) {
	controllerName, err := parseString(ing, IngressClassKey)
	return controllerName == ApplicationGatewayIngressClass, err
}

// IsSslRedirect for HTTP end points.
func IsSslRedirect(ing *v1beta1.Ingress) (bool, error) {
	return parseBool(ing, SslRedirectKey)
}

// BackendPathPrefix override path
func BackendPathPrefix(ing *v1beta1.Ingress) (string, error) {
	return parseString(ing, BackendPathPrefixKey)
}

func parseBool(ing *v1beta1.Ingress, name string) (bool, error) {
	val, ok := ing.Annotations[name]
	if ok {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return false, errors.NewInvalidAnnotationContent(name, val)
		}
		return boolVal, nil
	}
	return false, errors.ErrMissingAnnotations
}

func parseString(ing *v1beta1.Ingress, name string) (string, error) {
	val, ok := ing.Annotations[name]
	if ok {
		return val, nil
	}

	return "", errors.ErrMissingAnnotations
}

func parseInt32(ing *v1beta1.Ingress, name string) (int32, error) {
	val, ok := ing.Annotations[name]
	if ok {
		intVal, err := strconv.Atoi(val)
		if err == nil {
			int32Val := int32(intVal)
			return int32Val, nil
		}
		return 0, err
	}

	return 0, errors.ErrMissingAnnotations
}
