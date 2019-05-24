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

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
)

type backendIdentifier struct {
	serviceIdentifier
	Ingress *v1beta1.Ingress
	Rule    *v1beta1.IngressRule
	Path    *v1beta1.HTTPIngressPath
	Backend *v1beta1.IngressBackend
}

type serviceBackendPortPair struct {
	ServicePort int32
	BackendPort int32
}

type listenerIdentifier struct {
	FrontendPort int32
	HostName     string
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
var agPrefix = utils.GetEnv("APPGW_CONFIG_NAME_PREFIX", "", agPrefixValidator)

// create xxx -> xxxconfiguration mappings to contain all the information
type listenerAzConfig struct {
	Protocol                     network.ApplicationGatewayProtocol
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
	glog.Infof("Prop name %s with length %d is longer than %d characters; Transformed to %s", val, len(val), maxLen, finalVal)
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

func generateHTTPSettingsName(serviceName string, servicePort string, backendPortNo int32, ingress string) string {
	namePrefix := "bp-"
	return formatPropName(fmt.Sprintf("%s%s%v-%v-%v-%s", agPrefix, namePrefix, serviceName, servicePort, backendPortNo, ingress))
}

func generateProbeName(serviceName string, servicePort string, ingress string) string {
	namePrefix := "pb-"
	return formatPropName(fmt.Sprintf("%s%s%v-%v-%s", agPrefix, namePrefix, serviceName, servicePort, ingress))
}

func generateAddressPoolName(serviceName string, servicePort string, backendPortNo int32) string {
	namePrefix := "pool-"
	return formatPropName(fmt.Sprintf("%s%s%v-%v-bp-%v", agPrefix, namePrefix, serviceName, servicePort, backendPortNo))
}

func generateFrontendPortName(port int32) string {
	namePrefix := "fp-"
	return formatPropName(fmt.Sprintf("%s%s%v", agPrefix, namePrefix, port))
}

func generateListenerName(frontendListenerID listenerIdentifier) string {
	namePrefix := "fl-"
	return formatPropName(fmt.Sprintf("%s%s%v%v", agPrefix, namePrefix, formatHostname(frontendListenerID.HostName), frontendListenerID.FrontendPort))
}

func generateURLPathMapName(frontendListenerID listenerIdentifier) string {
	namePrefix := "url-"
	return formatPropName(fmt.Sprintf("%s%s%v%v", agPrefix, namePrefix, formatHostname(frontendListenerID.HostName), frontendListenerID.FrontendPort))
}

func generateRequestRoutingRuleName(frontendListenerID listenerIdentifier) string {
	namePrefix := "rr-"
	return formatPropName(fmt.Sprintf("%s%s%v%v", agPrefix, namePrefix, formatHostname(frontendListenerID.HostName), frontendListenerID.FrontendPort))
}

func generateSSLRedirectConfigurationName(namespace, ingress string) string {
	namePrefix := "sslr-"
	return formatPropName(fmt.Sprintf("%s%s%s-%s", agPrefix, namePrefix, namespace, ingress))
}

var defaultBackendHTTPSettingsName = fmt.Sprintf("%sdefaulthttpsetting", agPrefix)
var defaultBackendAddressPoolName = fmt.Sprintf("%sdefaultaddresspool", agPrefix)
var defaultProbeName = fmt.Sprintf("%sdefaultprobe", agPrefix)

func defaultBackendHTTPSettings(probeID string) network.ApplicationGatewayBackendHTTPSettings {
	defHTTPSettingsName := defaultBackendHTTPSettingsName
	defHTTPSettingsPort := int32(80)
	return network.ApplicationGatewayBackendHTTPSettings{
		Name: &defHTTPSettingsName,
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Protocol: network.HTTP,
			Port:     &defHTTPSettingsPort,
			Probe:    resourceRef(probeID),
		},
	}
}

func defaultProbe() network.ApplicationGatewayProbe {
	defProbeName := defaultProbeName
	defProtocol := network.HTTP
	defHost := "localhost"
	defPath := "/"
	defInterval := int32(30)
	defTimeout := int32(30)
	defUnHealthyCount := int32(3)
	return network.ApplicationGatewayProbe{
		Name: &defProbeName,
		ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
			Protocol:           defProtocol,
			Host:               &defHost,
			Path:               &defPath,
			Interval:           &defInterval,
			Timeout:            &defTimeout,
			UnhealthyThreshold: &defUnHealthyCount,
		},
	}
}

func defaultBackendAddressPool() network.ApplicationGatewayBackendAddressPool {
	defBackendAddressPool := defaultBackendAddressPoolName
	return network.ApplicationGatewayBackendAddressPool{
		Name: &defBackendAddressPool,
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &network.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: &[]network.ApplicationGatewayBackendAddress{},
		},
	}
}

func defaultFrontendListenerIdentifier() listenerIdentifier {
	return listenerIdentifier{
		FrontendPort: int32(80),
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
