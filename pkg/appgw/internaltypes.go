// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type backendIdentifier struct {
	serviceIdentifier
	ServicePort intstr.IntOrString
	Ingress     *v1beta1.Ingress
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

func generateHTTPSettingsName(serviceName string, servicePort string, backendPortNo int32, ingress string) string {
	return fmt.Sprintf("%s-%v-%v-bp-%v-%s", agPrefix, serviceName, servicePort, backendPortNo, ingress)
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

const defaultBackendHTTPSettingsName = agPrefix + "-defaulthttpsetting"
const defaultBackendAddressPoolName = agPrefix + "-defaultaddresspool"

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
