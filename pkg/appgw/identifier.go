// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
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

// AddressPoolID generates an ID for a backend address pool.
func (agw Identifier) AddressPoolID(poolName string) string {
	return agw.gatewayResourceID("backendAddressPools", poolName)
}

func (agw Identifier) frontendIPID(fipName string) string {
	return agw.gatewayResourceID("frontendIPConfigurations", fipName)
}

func (agw Identifier) frontendPortID(portName string) string {
	return agw.gatewayResourceID("frontendPorts", portName)
}

func (agw Identifier) sslCertificateID(certname string) string {
	return agw.gatewayResourceID("sslCertificates", certname)
}

// HTTPSettingsID generates an ID for App Gateway HTTP settings resource.
func (agw Identifier) HTTPSettingsID(settingsName string) string {
	return agw.gatewayResourceID("backendHttpSettingsCollection", settingsName)
}

func (agw Identifier) urlPathMapID(urlPathMapName string) string {
	return agw.gatewayResourceID("urlPathMaps", urlPathMapName)
}

func (agw Identifier) listenerID(listenerName string) string {
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

func (agw Identifier) requestRoutingRuleID(settingsName string) string {
	return agw.gatewayResourceID("requestRoutingRules", settingsName)
}

func resourceRef(id string) *n.SubResource {
	return &n.SubResource{ID: to.StringPtr(id)}
}
