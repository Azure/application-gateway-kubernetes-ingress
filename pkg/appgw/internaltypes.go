// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// A note on naming Application Gateway properties:
// A constraint on the App Gateway property names - these must begin and end with a word character or an underscore
// A word character is well defined here: https://docs.microsoft.com/en-us/dotnet/standard/base-types/character-classes-in-regular-expressions#WordCharacter

package appgw

import (
	"crypto/md5"
	"fmt"
	"math"
	"regexp"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/klog/v2"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

const (
	prefixHTTPSettings   = "bp"
	prefixProbe          = "pb"
	prefixPool           = "pool"
	prefixPort           = "fp"
	prefixListener       = "fl"
	prefixPathMap        = "url"
	prefixRoutingRule    = "rr"
	prefixRedirect       = "sslr"
	prefixPathRule       = "pr"
	prefixSslCertificate = "cert"
)

const (
	// MaxAllowedHostNames the maximum number of HostNames allowed for listener.
	MaxAllowedHostNames int = 5

	// WildcardSpecialCharacters are characters that are allowed for wildcard HostNames.
	WildcardSpecialCharacters = "*?"
)

type backendIdentifier struct {
	serviceIdentifier
	Ingress *v1beta1.Ingress
	Rule    *v1beta1.IngressRule
	Path    *v1beta1.HTTPIngressPath
	Backend *v1beta1.IngressBackend
}

type serviceBackendPortPair struct {
	ServicePort Port
	BackendPort Port
}

type listenerIdentifier struct {
	FrontendPort Port
	HostNames    [MaxAllowedHostNames]string
	UsePrivateIP bool
}

type serviceIdentifier struct {
	Namespace string
	Name      string
}

type secretIdentifier struct {
	Namespace string
	Name      string
}

// Max length for a property name is 80 characters. We hash w/ MD5 when length is > 80, which is 32 characters
var agPrefixValidator = regexp.MustCompile(`^[0-9a-zA-Z\-]{0,47}$`)
var agPrefix = environment.GetEnvironmentVariable("APPGW_CONFIG_NAME_PREFIX", "", agPrefixValidator)

// create xxx -> xxxconfiguration mappings to contain all the information
type listenerAzConfig struct {
	Protocol                     n.ApplicationGatewayProtocol
	Secret                       secretIdentifier
	SslRedirectConfigurationName string
	FirewallPolicy               string
}

// formatPropName ensures that the string generated is not longer than 80 characters.
func formatPropName(val string) string {
	// App Gateway property name cannot be longer than 80 characters
	maxLen := 80
	if len(val) <= maxLen {
		return val
	}
	hash := fmt.Sprintf("%x", md5.Sum([]byte(val)))
	separator := "-"
	prefix := val[0 : maxLen-len(hash)-len(separator)]
	finalVal := fmt.Sprintf("%s%s%s", prefix, separator, hash)
	klog.V(3).Infof("Prop name %s with length %d is longer than %d characters; Transformed to %s", val, len(val), maxLen, finalVal)
	return finalVal
}

func (s serviceIdentifier) serviceFullName() string {
	return fmt.Sprintf("%v-%v", s.Namespace, s.Name)
}

func (s serviceIdentifier) serviceKey() string {
	return fmt.Sprintf("%v/%v", s.Namespace, s.Name)
}

func (s secretIdentifier) secretKey() string {
	return fmt.Sprintf("%v/%v", s.Namespace, s.Name)
}

func (s secretIdentifier) secretFullName() string {
	// in case of referring ssl certificate from agic annotation
	if len(s.Namespace) == 0 {
		return s.Name
	}
	return fmt.Sprintf("%v%v-%v-%v", agPrefix, prefixSslCertificate, s.Namespace, s.Name)
}

func getResourceKey(namespace, name string) string {
	return formatPropName(fmt.Sprintf("%v/%v", namespace, name))
}

func generateHTTPSettingsName(serviceName string, servicePort string, backendPort Port, ingress string) string {
	return formatPropName(fmt.Sprintf("%s%s-%v-%v-%v-%s", agPrefix, prefixHTTPSettings, serviceName, servicePort, backendPort, ingress))
}

func generateProbeName(serviceName string, servicePort string, ingress *v1beta1.Ingress) string {
	return formatPropName(fmt.Sprintf("%s%s-%s-%v-%v-%s", agPrefix, prefixProbe, ingress.Namespace, serviceName, servicePort, ingress.Name))
}

func generateAddressPoolName(serviceName string, servicePort string, backendPort Port) string {
	return formatPropName(fmt.Sprintf("%s%s-%v-%v-bp-%v", agPrefix, prefixPool, serviceName, servicePort, backendPort))
}

func generateFrontendPortName(port Port) string {
	return formatPropName(fmt.Sprintf("%s%s-%v", agPrefix, prefixPort, port))
}

func generateListenerName(listenerID listenerIdentifier) string {
	return formatPropName(fmt.Sprintf("%s%s-%s", agPrefix, prefixListener, utils.GetHashCode(listenerID)))
}

func generateURLPathMapName(listenerID listenerIdentifier) string {
	return formatPropName(fmt.Sprintf("%s%s-%s", agPrefix, prefixPathMap, utils.GetHashCode(listenerID)))
}

func generateRequestRoutingRuleName(listenerID listenerIdentifier) string {
	return formatPropName(fmt.Sprintf("%s%s-%s", agPrefix, prefixRoutingRule, utils.GetHashCode(listenerID)))
}

func generateSSLRedirectConfigurationName(targetListener listenerIdentifier) string {
	return formatPropName(fmt.Sprintf("%s%s-%s", agPrefix, prefixRedirect, generateListenerName(targetListener)))
}

func generatePathRuleName(namespace, ingress string, ruleIdx, pathIdx int) string {
	return formatPropName(fmt.Sprintf("%s%s-%s-%s-rule-%d-path-%d", agPrefix, prefixPathRule, namespace, ingress, ruleIdx, pathIdx))
}

// DefaultBackendHTTPSettingsName is the name to be assigned to App Gateway's default HTTP settings resource.
var DefaultBackendHTTPSettingsName = fmt.Sprintf("%sdefaulthttpsetting", agPrefix)

// DefaultBackendAddressPoolName is the name to be assigned to App Gateway's default backend pool resource.
var DefaultBackendAddressPoolName = fmt.Sprintf("%sdefaultaddresspool", agPrefix)

func defaultProbeName(protocol n.ApplicationGatewayProtocol) string {
	return fmt.Sprintf("%sdefaultprobe-%s", agPrefix, protocol)
}

func defaultBackendHTTPSettings(appGWIdentifier Identifier, protocol n.ApplicationGatewayProtocol) n.ApplicationGatewayBackendHTTPSettings {
	defHTTPSettingsName := DefaultBackendHTTPSettingsName
	defHTTPSettingsPort := int32(80)
	return n.ApplicationGatewayBackendHTTPSettings{
		Name: &defHTTPSettingsName,
		ID:   to.StringPtr(appGWIdentifier.HTTPSettingsID(defHTTPSettingsName)),
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Protocol: protocol,
			Port:     &defHTTPSettingsPort,
			Probe:    resourceRef(appGWIdentifier.probeID(defaultProbeName(protocol))),

			// setting to default
			PickHostNameFromBackendAddress: to.BoolPtr(false),
			CookieBasedAffinity:            n.Disabled,
			RequestTimeout:                 to.Int32Ptr(30),
		},
	}
}

func defaultProbe(appGWIdentifier Identifier, protocol n.ApplicationGatewayProtocol) n.ApplicationGatewayProbe {
	defProbeName := defaultProbeName(protocol)
	defHost := "localhost"
	defPath := "/"
	defInterval := int32(30)
	defTimeout := int32(30)
	defUnHealthyCount := int32(3)
	return n.ApplicationGatewayProbe{
		Name: to.StringPtr(defProbeName),
		ID:   to.StringPtr(appGWIdentifier.probeID(defProbeName)),
		ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
			Protocol:           protocol,
			Host:               &defHost,
			Path:               &defPath,
			Interval:           &defInterval,
			Timeout:            &defTimeout,
			UnhealthyThreshold: &defUnHealthyCount,

			// setting to defaults
			Match:                               &n.ApplicationGatewayProbeHealthResponseMatch{},
			PickHostNameFromBackendHTTPSettings: to.BoolPtr(false),
			MinServers:                          to.Int32Ptr(0),
		},
	}
}

func defaultBackendAddressPool(appGWIdentifier Identifier) n.ApplicationGatewayBackendAddressPool {
	return n.ApplicationGatewayBackendAddressPool{
		Name: &DefaultBackendAddressPoolName,
		ID:   to.StringPtr(appGWIdentifier.AddressPoolID(DefaultBackendAddressPoolName)),
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: &[]n.ApplicationGatewayBackendAddress{},
		},
	}
}

func defaultFrontendListenerIdentifier() listenerIdentifier {
	return listenerIdentifier{
		FrontendPort: Port(80),
	}
}

func (listenerID *listenerIdentifier) setHostNames(hostNames []string) {
	hostnameCount := int(math.Min(float64(len(hostNames)), float64(MaxAllowedHostNames)))
	for i := 0; i < hostnameCount; i++ {
		listenerID.HostNames[i] = hostNames[i]
	}
}

// Returns the HostNames as a slice
func (listenerID *listenerIdentifier) getHostNames() []string {
	var hostNames []string

	for i := 0; i < len(listenerID.HostNames); i++ {
		if listenerID.HostNames[i] != "" {
			hostNames = append(hostNames, listenerID.HostNames[i])
		}
	}

	return hostNames
}

// getHostNameForProbes returns the first hostname which doesn't have special chars. To be used for probes.
func (listenerID *listenerIdentifier) getHostNameForProbes() *string {
	hostNames := listenerID.getHostNames()
	for _, hostName := range hostNames {
		if !strings.ContainsAny(hostName, WildcardSpecialCharacters) {
			return to.StringPtr(hostName)
		}
	}

	return nil
}
