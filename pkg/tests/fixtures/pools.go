// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	// DefaultBackendPoolName is a string constant.
	DefaultBackendPoolName = "defaultaddresspool"

	// BackendAddressPoolName1 is a string constant.
	BackendAddressPoolName1 = "BackendAddressPool-1"

	// BackendAddressPoolName2 is a string constant.
	BackendAddressPoolName2 = "BackendAddressPool-2"

	// BackendAddressPoolName3 is a string constant.
	BackendAddressPoolName3 = "BackendAddressPool-3"

	// IPAddress1 is a string constant.
	IPAddress1 = "1.2.3.4"

	// IPAddress2 is a string constant.
	IPAddress2 = "6.5.4.3"

	// IPAddress3 is a string constant.
	IPAddress3 = "99.95.94.93"
)

// GetDefaultBackendPool creates a new struct for use in unit tests.
func GetDefaultBackendPool() n.ApplicationGatewayBackendAddressPool {
	return n.ApplicationGatewayBackendAddressPool{
		Name: to.StringPtr(DefaultBackendPoolName),
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: &[]n.ApplicationGatewayBackendAddress{
				{
					IPAddress: to.StringPtr(IPAddress1),
				},
			},
		},
	}
}

// GetBackendPool1 creates a new struct for use in unit tests.
func GetBackendPool1() n.ApplicationGatewayBackendAddressPool {
	return n.ApplicationGatewayBackendAddressPool{
		Name: to.StringPtr(BackendAddressPoolName1),
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: &[]n.ApplicationGatewayBackendAddress{
				{
					IPAddress: to.StringPtr(IPAddress1),
				},
			},
		},
	}
}

// GetBackendPool2 creates a new struct for use in unit tests.
func GetBackendPool2() n.ApplicationGatewayBackendAddressPool {
	return n.ApplicationGatewayBackendAddressPool{
		Name: to.StringPtr(BackendAddressPoolName2),
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: &[]n.ApplicationGatewayBackendAddress{
				{
					IPAddress: to.StringPtr(IPAddress2),
				},
			},
		},
	}
}

// GetBackendPool3 creates a new struct for use in unit tests.
func GetBackendPool3() n.ApplicationGatewayBackendAddressPool {
	return n.ApplicationGatewayBackendAddressPool{
		Name: to.StringPtr(BackendAddressPoolName3),
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: &[]n.ApplicationGatewayBackendAddress{
				{
					IPAddress: to.StringPtr(IPAddress3),
				},
			},
		},
	}
}
