// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package annotations

import (
	"strconv"
	"strings"

	"github.com/knative/pkg/apis/istio/v1alpha3"
	networking "k8s.io/api/networking/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
)

const (
	// ApplicationGatewayPrefix defines the prefix for all keys associated with Application Gateway Ingress controller.
	ApplicationGatewayPrefix = "appgw.ingress.kubernetes.io"

	// BackendPathPrefixKey defines the key for Path which should be used as a prefix for all HTTP requests.
	// Null means no path will be prefixed. Default value is null.
	BackendPathPrefixKey = ApplicationGatewayPrefix + "/backend-path-prefix"

	// BackendHostNameKey defines the key for Host which should be used as when making a connection to the backend.
	// Null means Host specified in the request to Application Gateway is used to connect to the backend.
	BackendHostNameKey = ApplicationGatewayPrefix + "/backend-hostname"

	// HealthProbeHostKey defines the key for Host which should be used as a target for health probe.
	HealthProbeHostKey = ApplicationGatewayPrefix + "/health-probe-hostname"

	// HealthProbePortKey defines the key for port that should be used as a target for health probe.
	HealthProbePortKey = ApplicationGatewayPrefix + "/health-probe-port"

	// HealthProbePathKey defines the key for URL path which should be used as a target for health probe.
	HealthProbePathKey = ApplicationGatewayPrefix + "/health-probe-path"

	// HealthProbeStatusCodesKey defines status codes returned by the probe to be interpreted as healty service
	HealthProbeStatusCodesKey = ApplicationGatewayPrefix + "/health-probe-status-codes"

	// HealthProbeIntervalKey defines the probe interval in seconds
	HealthProbeIntervalKey = ApplicationGatewayPrefix + "/health-probe-interval"

	// HealthProbeTimeoutKey defines the probe timeout in seconds
	HealthProbeTimeoutKey = ApplicationGatewayPrefix + "/health-probe-timeout"

	// HealthProbeUnhealthyThresholdKey defines threshold for marking backend server as unhealthy
	HealthProbeUnhealthyThresholdKey = ApplicationGatewayPrefix + "/health-probe-unhealthy-threshold"

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

	// OverrideFrontendPortKey defines the key to define a custom fronend port
	OverrideFrontendPortKey = ApplicationGatewayPrefix + "/override-frontend-port"

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

	//DefaultIngressClass defines the default app gateway ingress value
	DefaultIngressClass = "azure/application-gateway"

	// FirewallPolicy is the key part of a key/value Ingress annotation.
	// The value of this is an ID of a Firewall Policy. The Firewall Policy must be already defined in Azure.
	// The policy will be attached to all URL paths declared in the annotated Ingress resource.
	FirewallPolicy = ApplicationGatewayPrefix + "/waf-policy-for-path"

	// AppGwSslCertificate indicates the name of ssl certificate installed by AppGw
	AppGwSslCertificate = ApplicationGatewayPrefix + "/appgw-ssl-certificate"

	// AppGwTrustedRootCertificate indicates the names of trusted root certificates
	// Multiple root certificates separated by comma, e.g. "cert1,cert2"
	AppGwTrustedRootCertificate = ApplicationGatewayPrefix + "/appgw-trusted-root-certificate"

	// RewriteRuleSetKey indicates the name of the rule set to overwrite HTTP headers.
	RewriteRuleSetKey = ApplicationGatewayPrefix + "/rewrite-rule-set"
)

var (
	// ApplicationGatewayIngressClass defines the value of the `IngressClassKey` and `IstioGatewayKey`
	// annotations that will tell the ingress controller whether it should act on this ingress resource or not.
	ApplicationGatewayIngressClass = DefaultIngressClass
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
func IsApplicationGatewayIngress(ing *networking.Ingress) (bool, error) {
	controllerName, err := parseString(ing, IngressClassKey)
	return controllerName == ApplicationGatewayIngressClass, err
}

// IsIstioGatewayIngress checks if this gateway should be handled by AGIC or not
func IsIstioGatewayIngress(gateway *v1alpha3.Gateway) (bool, error) {
	val, ok := gateway.Annotations[IstioGatewayKey]
	if ok {
		return val == ApplicationGatewayIngressClass, nil
	}
	return false, controllererrors.NewError(
		controllererrors.ErrorMissingAnnotation,
		"appgw.ingress.istio.io/v1alpha3 not set")
}

// IsSslRedirect for HTTP end points.
func IsSslRedirect(ing *networking.Ingress) (bool, error) {
	return parseBool(ing, SslRedirectKey)
}

// BackendPathPrefix override path
func BackendPathPrefix(ing *networking.Ingress) (string, error) {
	return parseString(ing, BackendPathPrefixKey)
}

// BackendHostName override hostname
func BackendHostName(ing *networking.Ingress) (string, error) {
	return parseString(ing, BackendHostNameKey)
}

// HealthProbeHostName probe hostname override
func HealthProbeHostName(ing *networking.Ingress) (string, error) {
	return parseString(ing, HealthProbeHostKey)
}

// HealthProbePort probe port override
func HealthProbePort(ing *networking.Ingress) (int32, error) {
	return parseInt32(ing, HealthProbePortKey)
}

// HealthProbePath probe path override
func HealthProbePath(ing *networking.Ingress) (string, error) {
	return parseString(ing, HealthProbePathKey)
}

// HealthProbeStatusCodes probe status codes
func HealthProbeStatusCodes(ing *networking.Ingress) ([]string, error) {
	value, err := parseString(ing, HealthProbeStatusCodesKey)
	if value != "" {
		codesArray := strings.Split(value, ",")
		for index, element := range codesArray {
			codesArray[index] = strings.TrimSpace(element)
		}
		return codesArray, err
	}

	return nil, err
}

// HealthProbeInterval probe interval
func HealthProbeInterval(ing *networking.Ingress) (int32, error) {
	return parseInt32(ing, HealthProbeIntervalKey)
}

// HealthProbeTimeout probe timeout
func HealthProbeTimeout(ing *networking.Ingress) (int32, error) {
	return parseInt32(ing, HealthProbeTimeoutKey)
}

// HealthProbeUnhealthyThreshold probe threshold
func HealthProbeUnhealthyThreshold(ing *networking.Ingress) (int32, error) {
	return parseInt32(ing, HealthProbeUnhealthyThresholdKey)
}

// GetAppGwSslCertificate refer to appgw installed certificate
func GetAppGwSslCertificate(ing *networking.Ingress) (string, error) {
	return parseString(ing, AppGwSslCertificate)
}

// GetAppGwTrustedRootCertificate refer to appgw installed root certificate
func GetAppGwTrustedRootCertificate(ing *networking.Ingress) (string, error) {
	return parseString(ing, AppGwTrustedRootCertificate)
}

// RequestTimeout provides value for request timeout on the backend connection
func RequestTimeout(ing *networking.Ingress) (int32, error) {
	return parseInt32(ing, RequestTimeoutKey)
}

// IsConnectionDraining provides whether connection draining is enabled or not.
func IsConnectionDraining(ing *networking.Ingress) (bool, error) {
	return parseBool(ing, ConnectionDrainingKey)
}

// ConnectionDrainingTimeout provides value for draining timeout for backends.
func ConnectionDrainingTimeout(ing *networking.Ingress) (int32, error) {
	return parseInt32(ing, ConnectionDrainingTimeoutKey)
}

// IsCookieBasedAffinity provides value to enable/disable cookie based affinity for client connection.
func IsCookieBasedAffinity(ing *networking.Ingress) (bool, error) {
	return parseBool(ing, CookieBasedAffinityKey)
}

// UsePrivateIP determines whether to use private IP with the ingress
func UsePrivateIP(ing *networking.Ingress) (bool, error) {
	return parseBool(ing, UsePrivateIPKey)
}

// OverrideFrontendPort determines whether to use a custom Frontend port
func OverrideFrontendPort(ing *networking.Ingress) (int32, error) {
	return parseInt32(ing, OverrideFrontendPortKey)
}

// BackendProtocol provides value for protocol to be used with the backend
func BackendProtocol(ing *networking.Ingress) (ProtocolEnum, error) {
	protocol, err := parseString(ing, BackendProtocolKey)
	if err != nil {
		return HTTP, err
	}

	if protocolEnum, ok := ProtocolEnumLookup[strings.ToLower(protocol)]; ok {
		return protocolEnum, nil
	}

	return HTTP, controllererrors.NewErrorf(controllererrors.ErrorInvalidContent,
		"annotation %v does not contain a valid value (%v)", BackendProtocolKey, protocol,
	)
}

// GetHostNameExtensions from a given ingress
func GetHostNameExtensions(ing *networking.Ingress) ([]string, error) {
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
func WAFPolicy(ing *networking.Ingress) (string, error) {
	return parseString(ing, FirewallPolicy)
}

// RewriteRuleSet name
func RewriteRuleSet(ing *networking.Ingress) (string, error) {
	return parseString(ing, RewriteRuleSetKey)
}

func parseBool(ing *networking.Ingress, name string) (bool, error) {
	if val, ok := ing.Annotations[name]; ok {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal, nil
		}
		return false, controllererrors.NewErrorf(controllererrors.ErrorInvalidContent,
			"annotation %v does not contain a valid value (%v)", name, val,
		)
	}
	return false, controllererrors.NewErrorf(
		controllererrors.ErrorMissingAnnotation,
		"%s is not set in Ingress %s/%s", name, ing.Namespace, ing.Name,
	)
}

func parseString(ing *networking.Ingress, name string) (string, error) {
	if val, ok := ing.Annotations[name]; ok {
		return val, nil
	}
	return "", controllererrors.NewErrorf(
		controllererrors.ErrorMissingAnnotation,
		"%s is not set in Ingress %s/%s", name, ing.Namespace, ing.Name,
	)
}

func parseInt32(ing *networking.Ingress, name string) (int32, error) {
	if val, ok := ing.Annotations[name]; ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return int32(intVal), nil
		}
		return 0, controllererrors.NewErrorf(controllererrors.ErrorInvalidContent,
			"annotation %v does not contain a valid value (%v)", name, val,
		)
	}

	return 0, controllererrors.NewErrorf(
		controllererrors.ErrorMissingAnnotation,
		"%s is not set in Ingress %s/%s", name, ing.Namespace, ing.Name,
	)
}
