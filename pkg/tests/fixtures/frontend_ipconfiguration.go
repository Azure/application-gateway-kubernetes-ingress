// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	"github.com/Azure/go-autorest/autorest/to"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
)

const (
	// PublicIPName is a string constant
	PublicIPName = "PublicIP"

	// PrivateIPName is a string constant
	PrivateIPName = "PrivateIP"
)

// GetPublicIPConfiguration get a frontend IP configuration with public ip reference
func GetPublicIPConfiguration() n.ApplicationGatewayFrontendIPConfiguration {
	return n.ApplicationGatewayFrontendIPConfiguration{
		Name: to.StringPtr(PublicIPName),
		Etag: to.StringPtr("xx2"),
		Type: to.StringPtr("xx1"),
		ID:   to.StringPtr(PublicIPName),
		ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &n.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
			PrivateIPAddress: nil,
			PublicIPAddress: &n.SubResource{
				ID: to.StringPtr("xyz"),
			},
		},
	}
}

// GetPrivateIPConfiguration get a frontend IP configuration with private reference
func GetPrivateIPConfiguration() n.ApplicationGatewayFrontendIPConfiguration {
	return n.ApplicationGatewayFrontendIPConfiguration{
		Name: to.StringPtr(PrivateIPName),
		Etag: to.StringPtr("yy2"),
		Type: to.StringPtr("yy1"),
		ID:   to.StringPtr(PrivateIPName),
		ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &n.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
			PrivateIPAddress: to.StringPtr("abc"),
			PublicIPAddress:  nil,
		},
	}
}
