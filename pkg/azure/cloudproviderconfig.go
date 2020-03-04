// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// CloudProviderConfig represents the Azure context file
type CloudProviderConfig struct {
	Cloud                   string `json:"cloud"`
	TenantID                string `json:"tenantId"`
	SubscriptionID          string `json:"subscriptionId"`
	ClientID                string `json:"aadClientId"`
	ClientSecret            string `json:"aadClientSecret"`
	ResourceGroup           string `json:"resourceGroup"`
	Region                  string `json:"location"`
	VNetName                string `json:"vnetName"`
	VNetResourceGroup       string `json:"vnetResourceGroup"`
	RouteTableName          string `json:"routeTableName"`
	RouteTableResourceGroup string `json:"routeTableResourceGroup"`
	UserAssignedIdentityID  string `json:"userAssignedIdentityID"`
}

// NewCloudProviderConfig returns an CloudProviderConfig struct from file path
func NewCloudProviderConfig(path string) (*CloudProviderConfig, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Reading Az Context file %q failed: %v", path, err)
	}

	// Unmarshal the authentication file.
	var context CloudProviderConfig
	if err := json.Unmarshal(b, &context); err != nil {
		return nil, err
	}

	if context.VNetResourceGroup == "" {
		context.VNetResourceGroup = context.ResourceGroup
	}
	if context.RouteTableResourceGroup == "" {
		context.RouteTableResourceGroup = context.ResourceGroup
	}

	return &context, nil
}
