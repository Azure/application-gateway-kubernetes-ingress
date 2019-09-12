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
	"regexp"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
)

const (
	prefixHTTPSettings = "bp"
	prefixProbe        = "pb"
	prefixPool         = "pool"
	prefixPort         = "fp"
	prefixListener     = "fl"
	prefixPathMap      = "url"
	prefixRoutingRule  = "rr"
	prefixRedirect     = "sslr"
	prefixPathRule     = "pr"
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
	HostName     string
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
	glog.V(3).Infof("Prop name %s with length %d is longer than %d characters; Transformed to %s", val, len(val), maxLen, finalVal)
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
	return fmt.Sprintf("%v-%v", s.Namespace, s.Name)
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
	if listenerID.UsePrivateIP {
		return formatPropName(fmt.Sprintf("%s%s-%v%v-privateip", agPrefix, prefixListener, formatHostname(listenerID.HostName), listenerID.FrontendPort))
	}
	return formatPropName(fmt.Sprintf("%s%s-%v%v", agPrefix, prefixListener, formatHostname(listenerID.HostName), listenerID.FrontendPort))
}

func generateURLPathMapName(listenerID listenerIdentifier) string {
	return formatPropName(fmt.Sprintf("%s%s-%v%v", agPrefix, prefixPathMap, formatHostname(listenerID.HostName), listenerID.FrontendPort))
}

func generateRequestRoutingRuleName(listenerID listenerIdentifier) string {
	return formatPropName(fmt.Sprintf("%s%s-%v%v", agPrefix, prefixRoutingRule, formatHostname(listenerID.HostName), listenerID.FrontendPort))
}

func generateSSLRedirectConfigurationName(targetListener listenerIdentifier) string {
	return formatPropName(fmt.Sprintf("%s%s-%s", agPrefix, prefixRedirect, generateListenerName(targetListener)))
}

func generatePathRuleName(namespace, ingress, suffix string) string {
	return formatPropName(fmt.Sprintf("%s%s-%s-%s-%s", agPrefix, prefixPathRule, namespace, ingress, suffix))
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
		HostName:     "",
	}
}

// formatHostname formats the hostname, which could be an empty string.
func formatHostname(hostName string) string {
	// Hostname could be empty.
	if hostName == "" {
		return ""
	}
	// Hostname is NOT empty - prefix it with a dash
	return fmt.Sprintf("%s-", hostName)
}
