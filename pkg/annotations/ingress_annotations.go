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

	// OverrideBackendHostNameKey defines the key to enable/disable host name overriding.
	OverrideBackendHostNameKey = ApplicationGatewayPrefix + "/override-backend-host"

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

	// CookieBasedAffinityDistinctNameKey defines the key to enable/disable distinct cookie names per backend for client connection.
	CookieBasedAffinityDistinctNameKey = ApplicationGatewayPrefix + "/cookie-based-affinity-distinct-name"

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

	// FirewallPolicy is the key part of a key/value Ingress annotation.
	// The value of this is an ID of a Firewall Policy. The Firewall Policy must be already defined in Azure.
	// The policy will be attached to all URL paths declared in the annotated Ingress resource.
	FirewallPolicy = ApplicationGatewayPrefix + "/waf-policy-for-path"

	// AppGwSslCertificate indicates the name of ssl certificate installed by AppGw
	AppGwSslCertificate = ApplicationGatewayPrefix + "/appgw-ssl-certificate"

	// AppGwSslProfile indicates the name of the ssl profile installed by AppGw
	AppGwSslProfile = ApplicationGatewayPrefix + "/appgw-ssl-profile"

	// AppGwTrustedRootCertificate indicates the names of trusted root certificates
	// Multiple root certificates separated by comma, e.g. "cert1,cert2"
	AppGwTrustedRootCertificate = ApplicationGatewayPrefix + "/appgw-trusted-root-certificate"

	// RewriteRuleSetKey indicates the name of the rule set to overwrite HTTP headers.
	RewriteRuleSetKey = ApplicationGatewayPrefix + "/rewrite-rule-set"

	// RewriteRuleSetCustomResourceKey indicates the name of the rule set CRD to use for header CRD and URL Config.
	RewriteRuleSetCustomResourceKey = ApplicationGatewayPrefix + "/rewrite-rule-set-custom-resource"

	// RequestRoutingRulePriority indicates the priority of the Request Routing Rules.
	RequestRoutingRulePriority = ApplicationGatewayPrefix + "/rule-priority"
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

// IngressClass returns ingress class annotation value if set
func IngressClass(ing *networking.Ingress) (string, error) {
	return parseString(ing, IngressClassKey)
}

// IngressClass returns istio ingress class annotation value if set
func IstioGatewayIngressClass(gateway *v1alpha3.Gateway) (string, error) {
	val, ok := gateway.Annotations[IstioGatewayKey]
	if ok {
		return val, nil
	}

	return "", controllererrors.NewErrorf(
		controllererrors.ErrorMissingAnnotation,
		"%s is not set in Ingress %s/%s", IstioGatewayKey, gateway.Namespace, gateway.Name,
	)
}

// IsSslRedirect for HTTP end points.
func IsSslRedirect(ing *networking.Ingress) (bool, error) {
	return parseBool(ing, SslRedirectKey)
}

// BackendPathPrefix override path
func BackendPathPrefix(ing *networking.Ingress) (string, error) {
	return parseString(ing, BackendPathPrefixKey)
}

// BackendHostName specific domain name to override the backend hostname with
func BackendHostName(ing *networking.Ingress) (string, error) {
	return parseString(ing, BackendHostNameKey)
}

// OverrideBackendHostName whether to override the backend hostname or not
func OverrideBackendHostName(ing *networking.Ingress) (string, error) {
	return parseBool(ing, OverrideBackendHostNameKey)
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

// GetAppGwSslProfile refer to appgw installed certificate
func GetAppGwSslProfile(ing *networking.Ingress) (string, error) {
	return parseString(ing, AppGwSslProfile)
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

// IsCookieBasedAffinityDistinctName provides value to enable/disable distinct cookie name based affinity for client connection.
func IsCookieBasedAffinityDistinctName(ing *networking.Ingress) (bool, error) {
	return parseBool(ing, CookieBasedAffinityDistinctNameKey)
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
			trimmed := strings.TrimSpace(hostname)
			if len(trimmed) > 0 {
				hostnames = append(hostnames, trimmed)
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

// RewriteRuleSetCustomResource name
func RewriteRuleSetCustomResource(ing *networking.Ingress) (string, error) {
	return parseString(ing, RewriteRuleSetCustomResourceKey)
}

// GetRequestRoutingRulePriority gets the request routing rule priority
func GetRequestRoutingRulePriority(ing *networking.Ingress) (*int32, error) {
	min := int32(1)
	max := int32(20000)
	val, err := parseInt32(ing, RequestRoutingRulePriority)
	if err == nil {
		if val >= min && val <= max {
			return &val, nil
		} else {
			val = 0
			return &val, controllererrors.NewErrorf(controllererrors.ErrorInvalidContent,
				"Priority must be a value from %d to %d", min, max)
		}
	}

	return nil, err
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
