// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package annotations

import (
	"strconv"
	"strings"

	"github.com/knative/pkg/apis/istio/v1alpha3"
	"k8s.io/api/extensions/v1beta1"
)

const (
	// ApplicationGatewayPrefix defines the prefix for all keys associated with Application Gateway Ingress controller.
	ApplicationGatewayPrefix = "appgw.ingress.kubernetes.io"

	// BackendPathPrefixKey defines the key for Path which should be used as a prefix for all HTTP requests.
	// Null means no path will be prefixed. Default value is null.
	BackendPathPrefixKey = ApplicationGatewayPrefix + "/backend-path-prefix"

	// CookieBasedAffinityKey defines the key to enable/disable cookie based affinity for client connection.
	CookieBasedAffinityKey = ApplicationGatewayPrefix + "/cookie-based-affinity"

	// RequestTimeoutKey defines the request timeout to the backend.
	RequestTimeoutKey = ApplicationGatewayPrefix + "/request-timeout"

	// ConnectionDrainingKey defines the key to enable/disable connection draining.
	ConnectionDrainingKey = ApplicationGatewayPrefix + "/connection-draining"

	// ConnectionDrainingTimeoutKey defines the drain timeout for the backends.
	ConnectionDrainingTimeoutKey = ApplicationGatewayPrefix + "/connection-draining-timeout"

	// SslRedirectKey defines the key for defining with SSL redirect should be turned on for an HTTP endpoint.
	SslRedirectKey = ApplicationGatewayPrefix + "/ssl-redirect"

	// UsePrivateIPKey defines the key to determine whether to use private ip with the ingress.
	UsePrivateIPKey = ApplicationGatewayPrefix + "/use-private-ip"

	// BackendProtocolKey defines the key to determine whether to use private ip with the ingress.
	BackendProtocolKey = ApplicationGatewayPrefix + "/backend-protocol"

	// HostNameExtensionKey defines the key to add multiple hostnames to ingress rules including wildcard hostnames
	// annotation will be appgw.ingress.kubernetes.io/hostname-extension : "hostname1, hostname2"
	// The extended hostnames will be appended to ingress host for a rule if specified
	HostNameExtensionKey = ApplicationGatewayPrefix + "/hostname-extension"

	// IngressClassKey defines the key of the annotation which needs to be set in order to specify
	// that this is an ingress resource meant for the application gateway ingress controller.
	IngressClassKey = "kubernetes.io/ingress.class"

	// IstioGatewayKey defines the key of the annotation which needs to be set in order to specify
	// that this is a gateway meant for the application gateway ingress controller.
	IstioGatewayKey = "appgw.ingress.istio.io/v1alpha3"

	// ApplicationGatewayIngressClass defines the value of the `IngressClassKey` and `IstioGatewayKey`
	// annotations that will tell the ingress controller whether it should act on this ingress resource or not.
	ApplicationGatewayIngressClass = "azure/application-gateway"

	// FirewallPolicy is the key part of a key/value Ingress annotation.
	// The value of this is an ID of a Firewall Policy. The Firewall Policy must be already defined in Azure.
	// The policy will be attached to all URL paths declared in the annotated Ingress resource.
	FirewallPolicy = ApplicationGatewayPrefix + "/waf-policy-for-path"
)

// ProtocolEnum is the type for protocol
type ProtocolEnum int

const (
	// HTTP is enum for http protocol
	HTTP ProtocolEnum = iota + 1

	// HTTPS is enum for https protocol
	HTTPS
)

// ProtocolEnumLookup is a reverse map of the EventType enums; used for logging purposes
var ProtocolEnumLookup = map[string]ProtocolEnum{
	"http":  HTTP,
	"https": HTTPS,
}

// IsApplicationGatewayIngress checks if the Ingress resource can be handled by the Application Gateway ingress controller.
func IsApplicationGatewayIngress(ing *v1beta1.Ingress) (bool, error) {
	controllerName, err := parseString(ing, IngressClassKey)
	return controllerName == ApplicationGatewayIngressClass, err
}

// IsIstioGatewayIngress checks if this gateway should be handled by AGIC or not
func IsIstioGatewayIngress(gateway *v1alpha3.Gateway) (bool, error) {
	val, ok := gateway.Annotations[IstioGatewayKey]
	if ok {
		return val == ApplicationGatewayIngressClass, nil
	}
	return false, ErrMissingAnnotations
}

// IsSslRedirect for HTTP end points.
func IsSslRedirect(ing *v1beta1.Ingress) (bool, error) {
	return parseBool(ing, SslRedirectKey)
}

// BackendPathPrefix override path
func BackendPathPrefix(ing *v1beta1.Ingress) (string, error) {
	return parseString(ing, BackendPathPrefixKey)
}

// RequestTimeout provides value for request timeout on the backend connection
func RequestTimeout(ing *v1beta1.Ingress) (int32, error) {
	return parseInt32(ing, RequestTimeoutKey)
}

// IsConnectionDraining provides whether connection draining is enabled or not.
func IsConnectionDraining(ing *v1beta1.Ingress) (bool, error) {
	return parseBool(ing, ConnectionDrainingKey)
}

// ConnectionDrainingTimeout provides value for draining timeout for backends.
func ConnectionDrainingTimeout(ing *v1beta1.Ingress) (int32, error) {
	return parseInt32(ing, ConnectionDrainingTimeoutKey)
}

// IsCookieBasedAffinity provides value to enable/disable cookie based affinity for client connection.
func IsCookieBasedAffinity(ing *v1beta1.Ingress) (bool, error) {
	return parseBool(ing, CookieBasedAffinityKey)
}

// UsePrivateIP determines whether to use private IP with the ingress
func UsePrivateIP(ing *v1beta1.Ingress) (bool, error) {
	return parseBool(ing, UsePrivateIPKey)
}

// BackendProtocol provides value for protocol to be used with the backend
func BackendProtocol(ing *v1beta1.Ingress) (ProtocolEnum, error) {
	protocol, err := parseString(ing, BackendProtocolKey)
	if err != nil {
		return HTTP, err
	}

	if protocolEnum, ok := ProtocolEnumLookup[strings.ToLower(protocol)]; ok {
		return protocolEnum, nil
	}

	return HTTP, NewInvalidAnnotationContent(BackendProtocolKey, protocol)
}

// GetHostNameExtensions from a given ingress
func GetHostNameExtensions(ing *v1beta1.Ingress) ([]string, error) {
	val, err := parseString(ing, HostNameExtensionKey)
	if err == nil {
		var hostnames []string
		for _, hostname := range strings.Split(val, ",") {
			if len(hostname) > 0 {
				hostnames = append(hostnames, strings.TrimSpace(hostname))
			}
		}
		return hostnames, nil
	}

	return nil, err
}

// WAFPolicy override path
func WAFPolicy(ing *v1beta1.Ingress) (string, error) {
	return parseString(ing, FirewallPolicy)
}

func parseBool(ing *v1beta1.Ingress, name string) (bool, error) {
	if val, ok := ing.Annotations[name]; ok {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal, nil
		}
		return false, NewInvalidAnnotationContent(name, val)
	}
	return false, ErrMissingAnnotations
}

func parseString(ing *v1beta1.Ingress, name string) (string, error) {
	if val, ok := ing.Annotations[name]; ok {
		return val, nil
	}
	return "", ErrMissingAnnotations
}

func parseInt32(ing *v1beta1.Ingress, name string) (int32, error) {
	if val, ok := ing.Annotations[name]; ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return int32(intVal), nil
		}
		return 0, NewInvalidAnnotationContent(name, val)
	}

	return 0, ErrMissingAnnotations
}
