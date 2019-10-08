// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"

// GetGatewayFunc is a function type
type GetGatewayFunc func() (n.ApplicationGateway, error)

// UpdateGatewayFunc is a function type
type UpdateGatewayFunc func(*n.ApplicationGateway) error

// DeployGatewayFunc is a function type
type DeployGatewayFunc func(string) error

// GetPublicIPFunc is a function type
type GetPublicIPFunc func(string) (n.PublicIPAddress, error)

// FakeAzClient is a fake struct for AzClient
type FakeAzClient struct {
	GetGatewayFunc
	UpdateGatewayFunc
	DeployGatewayFunc
	GetPublicIPFunc
}

// NewFakeAzClient returns a fake Azure Client
func NewFakeAzClient() *FakeAzClient {
	return &FakeAzClient{}
}

// GetGateway runs GetGatewayFunc and return a gateway
func (az *FakeAzClient) GetGateway() (n.ApplicationGateway, error) {
	if az.GetGatewayFunc != nil {
		return az.GetGatewayFunc()
	}
	return n.ApplicationGateway{}, nil
}

// UpdateGateway runs UpdateGatewayFunc and return a gateway
func (az *FakeAzClient) UpdateGateway(appGwObj *n.ApplicationGateway) (err error) {
	if az.UpdateGatewayFunc != nil {
		return az.UpdateGatewayFunc(appGwObj)
	}
	return nil
}

// DeployGatewayWithSubnet runs DeployGatewayFunc
func (az *FakeAzClient) DeployGatewayWithSubnet(subnetID string) (err error) {
	if az.DeployGatewayFunc != nil {
		return az.DeployGatewayFunc(subnetID)
	}
	return nil
}

// DeployGatewayWithVnet runs DeployGatewayFunc
func (az *FakeAzClient) DeployGatewayWithVnet(resourceGroupName ResourceGroup, vnetName ResourceName, subnetPrefix string) (err error) {
	if az.DeployGatewayFunc != nil {
		return az.DeployGatewayFunc(subnetPrefix)
	}
	return nil
}

// GetPublicIP runs GetPublicIPFunc
func (az *FakeAzClient) GetPublicIP(resourceID string) (n.PublicIPAddress, error) {
	if az.GetPublicIPFunc != nil {
		return az.GetPublicIPFunc(resourceID)
	}
	return n.PublicIPAddress{}, nil
}
