// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

// Identifier is identifier for a specific Application Gateway
type Identifier struct {
	SubscriptionID string
	ResourceGroup  string
	AppGwName      string
}

func (agw Identifier) resourceID(provider string, resourceKind string, resourcePath string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s",
		agw.SubscriptionID, agw.ResourceGroup, provider, resourceKind, resourcePath)
}

func (agw Identifier) gatewayResourceID(subResourceKind string, resourceName string) string {
	resourcePath := fmt.Sprintf("%s/%s/%s", agw.AppGwName, subResourceKind, resourceName)
	return agw.resourceID("Microsoft.Network", "applicationGateways", resourcePath)
}

func (agw Identifier) addressPoolID(poolName string) string {
	return agw.gatewayResourceID("backendAddressPools", poolName)
}

func (agw Identifier) frontendIPID(fipName string) string {
	return agw.gatewayResourceID("frontEndIPConfigurations", fipName)
}

func (agw Identifier) frontendPortID(portName string) string {
	return agw.gatewayResourceID("frontEndPorts", portName)
}

func (agw Identifier) sslCertificateID(certname string) string {
	return agw.gatewayResourceID("sslCertificates", certname)
}

func (agw Identifier) httpSettingsID(settingsName string) string {
	return agw.gatewayResourceID("backendHttpSettingsCollection", settingsName)
}

func (agw Identifier) urlPathMapID(urlPathMapName string) string {
	return agw.gatewayResourceID("urlPathMaps", urlPathMapName)
}

func (agw Identifier) httpListenerID(listenerName string) string {
	return agw.gatewayResourceID("httpListeners", listenerName)
}

func (agw Identifier) redirectConfigurationID(configurationName string) string {
	return agw.gatewayResourceID("redirectConfigurations", configurationName)
}

func (agw Identifier) probeID(probeName string) string {
	return agw.gatewayResourceID("probes", probeName)
}

func (agw Identifier) subnetID(vnetName string, subnetName string) string {
	resourcePath := fmt.Sprintf("%s/subnets/%s", vnetName, subnetName)
	return agw.resourceID("Microsoft.Network", "virtualNetworks", resourcePath)
}

func (agw Identifier) publicIPID(publicIPName string) string {
	return agw.resourceID("Microsoft.Network", "publicIPAddresses", publicIPName)
}

func resourceRef(id string) *network.SubResource {
	return &network.SubResource{ID: to.StringPtr(id)}
}
