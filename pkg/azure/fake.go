// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"

type fakeAzClient struct {
}

// NewFakeAzClient returns a fake Azure Client
func NewFakeAzClient() AzClient {
	return &fakeAzClient{}
}

func (az *fakeAzClient) GetGateway() (n.ApplicationGateway, error) {
	return n.ApplicationGateway{}, nil
}

func (az *fakeAzClient) UpdateGateway(appGwObj *n.ApplicationGateway) (err error) {
	return nil
}

func (az *fakeAzClient) DeployGateway(subnetID string) (err error) {
	return nil
}

func (az *fakeAzClient) GetPublicIP(resourceID string) (n.PublicIPAddress, error) {
	return n.PublicIPAddress{}, nil
}
