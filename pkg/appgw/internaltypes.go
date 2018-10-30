// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type backendIdentifier struct {
	serviceIdentifier
	ServicePort intstr.IntOrString
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

// create xxx -> xxxconfiguration mappings to contain all the information
type frontendListenerAzureConfig struct {
	Protocol network.ApplicationGatewayProtocol
	Secret   secretIdentifier
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

func generateHTTPSettingsName(serviceName string, servicePort string, backendPortNo int32) string {
	return fmt.Sprintf("k8s-%v-%v-bp-%v", serviceName, servicePort, backendPortNo)
}

func generateAddressPoolName(serviceName string, servicePort string, backendPortNo int32) string {
	return fmt.Sprintf("k8s-%v-%v-bp-%v-pool", serviceName, servicePort, backendPortNo)
}

func generateFrontendPortName(port int32) string {
	return fmt.Sprintf("k8s-fp-%v", port)
}

func generateHTTPListenerName(frontendListenerID frontendListenerIdentifier) string {
	return fmt.Sprintf("k8s-%v-%v-fl", frontendListenerID.HostName, frontendListenerID.FrontendPort)
}

func generateURLPathMapName(frontendListenerID frontendListenerIdentifier) string {
	return fmt.Sprintf("k8s-%v-%v-url", frontendListenerID.HostName, frontendListenerID.FrontendPort)
}

func generateRequestRoutingRuleName(frontendListenerID frontendListenerIdentifier) string {
	return fmt.Sprintf("k8s-%v-%v-rr", frontendListenerID.HostName, frontendListenerID.FrontendPort)
}

const defaultBackendHTTPSettingsName = "k8s-defaulthttpsetting"
const defaultBackendAddressPoolName = "k8s-defaultaddresspool"

func defaultBackendHTTPSettings() network.ApplicationGatewayBackendHTTPSettings {
	defHTTPSettingsName := defaultBackendHTTPSettingsName
	defHTTPSettingsPort := int32(80)
	return network.ApplicationGatewayBackendHTTPSettings{
		Name: &defHTTPSettingsName,
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Protocol: network.HTTP,
			Port:     &defHTTPSettingsPort,
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
