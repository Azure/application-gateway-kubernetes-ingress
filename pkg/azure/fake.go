// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"github.com/Azure/go-autorest/autorest"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
)

// GetGatewayFunc is a function type
type GetGatewayFunc func() (n.ApplicationGateway, error)

// UpdateGatewayFunc is a function type
type UpdateGatewayFunc func(*n.ApplicationGateway) error

// DeployGatewayFunc is a function type
type DeployGatewayFunc func(string) error

// GetPublicIPFunc is a function type
type GetPublicIPFunc func(string) (n.PublicIPAddress, error)

// ApplyRouteTableFunc is a function type
type ApplyRouteTableFunc func(string, string) error

// CheckAccessFunc is a function type
type CheckAccessFunc func() (bool, error)

// FakeAzClient is a fake struct for AzClient
type FakeAzClient struct {
	CheckAccessFunc
	GetGatewayFunc
	UpdateGatewayFunc
	DeployGatewayFunc
	GetPublicIPFunc
	ApplyRouteTableFunc
}

// NewFakeAzClient returns a fake Azure Client
func NewFakeAzClient() *FakeAzClient {
	return &FakeAzClient{}
}

// SetAuthorizer is an empty function
func (az *FakeAzClient) SetAuthorizer(authorizer autorest.Authorizer) {
}

// CheckAccess runs CheckAccessFunc and returns if AGIC has access to the gateway or not
func (az *FakeAzClient) CheckAccess(string, RoleDefinition) (bool, error) {
	if az.CheckAccessFunc != nil {
		return az.CheckAccessFunc()
	}
	return true, nil
}

// WaitForAccess runs CheckAccessFunc in a loop and returns when CheckAccessFunc is true
func (az *FakeAzClient) WaitForAccess(string, RoleDefinition) {
	if az.CheckAccessFunc != nil {
		for {
			hasAccess, _ := az.CheckAccessFunc()
			if hasAccess {
				return
			}
		}
	}

	return
}

// GetGateway runs GetGatewayFunc and return a gateway
func (az *FakeAzClient) GetGateway() (n.ApplicationGateway, error) {
	if az.GetGatewayFunc != nil {
		return az.GetGatewayFunc()
	}
	return n.ApplicationGateway{}, nil
}

// WaitForGetAccessOnGateway runs GetGatewayFunc until it returns a gateway
func (az *FakeAzClient) WaitForGetAccessOnGateway() error {
	if az.GetGatewayFunc != nil {
		for {
			_, err := az.GetGatewayFunc()
			if err == nil {
				return nil
			}
		}
	}

	return nil
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
func (az *FakeAzClient) DeployGatewayWithVnet(resourceGroupName ResourceGroup, vnetName ResourceName, subnetName ResourceName, subnetPrefix string) (err error) {
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

// ApplyRouteTable runs ApplyRouteTableFunc
func (az *FakeAzClient) ApplyRouteTable(subnetID string, routeTableID string) error {
	if az.ApplyRouteTableFunc != nil {
		return az.ApplyRouteTableFunc(subnetID, routeTableID)
	}
	return nil
}
