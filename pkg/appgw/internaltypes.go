// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"crypto/md5"
	"fmt"

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

type frontendListenerIdentifier struct {
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

const (
	agPrefix = "k8s-ag-ingress"
)

// create xxx -> xxxconfiguration mappings to contain all the information
type frontendListenerAzureConfig struct {
	Protocol                     network.ApplicationGatewayProtocol
	Secret                       secretIdentifier
	SslRedirectConfigurationName string
}

// governor ensures that the string generated is not longer than 80 characters.
func governor(val string) string {
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
	return fmt.Sprintf("%v/%v", namespace, name)
}

func generateHTTPSettingsName(serviceName string, servicePort string, backendPortNo int32, ingress string) string {
	return fmt.Sprintf("%s-%v-%v-bp-%v-%s", agPrefix, serviceName, servicePort, backendPortNo, ingress)
}

func generateProbeName(serviceName string, servicePort string, ingress string) string {
	return governor(fmt.Sprintf("%s-%v-%v-pb-%s", agPrefix, serviceName, servicePort, ingress))
}

func generateAddressPoolName(serviceName string, servicePort string, backendPortNo int32) string {
	return fmt.Sprintf("%s-%v-%v-bp-%v-pool", agPrefix, serviceName, servicePort, backendPortNo)
}

func generateFrontendPortName(port int32) string {
	return fmt.Sprintf("%s-fp-%v", agPrefix, port)
}

func generateHTTPListenerName(frontendListenerID frontendListenerIdentifier) string {
	return fmt.Sprintf("%s-%v-%v-fl", agPrefix, frontendListenerID.HostName, frontendListenerID.FrontendPort)
}

func generateURLPathMapName(frontendListenerID frontendListenerIdentifier) string {
	return fmt.Sprintf("%s-%v-%v-url", agPrefix, frontendListenerID.HostName, frontendListenerID.FrontendPort)
}

func generateRequestRoutingRuleName(frontendListenerID frontendListenerIdentifier) string {
	return fmt.Sprintf("%s-%v-%v-rr", agPrefix, frontendListenerID.HostName, frontendListenerID.FrontendPort)
}

func generateSSLRedirectConfigurationName(namespace, ingress string) string {
	return fmt.Sprintf("%s-%s-%s-sslr", agPrefix, namespace, ingress)
}

const defaultBackendHTTPSettingsName = agPrefix + "-defaulthttpsetting"
const defaultBackendAddressPoolName = agPrefix + "-defaultaddresspool"
const defaultProbeName = agPrefix + "-defaultprobe"

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

func defaultFrontendListenerIdentifier() frontendListenerIdentifier {
	return frontendListenerIdentifier{
		FrontendPort: int32(80),
		HostName:     "",
	}
}
