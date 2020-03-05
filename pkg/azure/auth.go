// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/glog"
)

func getAuthorizer(authLocation string, useManagedidentity bool, cpConfig *CloudProviderConfig, cloudName string) (autorest.Authorizer, error) {
	// Authorizer logic:
	// 1. If User provided authLocation, then use the file.
	// 2. If User provided a managed identity in ex: helm config, then use Environment
	// 3. If User provided nothing and CloudProviderConfig has value, then use CloudProviderConfig
	// 4. Fall back to environment
	if authLocation != "" {
		glog.V(1).Infof("Creating authorizer from file referenced by environment variable AZURE_AUTH_LOCATION: %s", authLocation)
		return auth.NewAuthorizerFromFile(n.DefaultBaseURI)
	}
	if !useManagedidentity && cpConfig != nil {
		glog.V(1).Info("Creating authorizer using Cluster Service Principal.")
		credAuthorizer := auth.NewClientCredentialsConfig(cpConfig.ClientID, cpConfig.ClientSecret, cpConfig.TenantID)
		return credAuthorizer.Authorizer()
	}

	glog.V(1).Info("Creating authorizer from Azure Managed Service Identity")
	setting, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil, err
	}

	setting.Environment, err = azure.EnvironmentFromName(cloudName)
	if err != nil {
		return nil, err
	}

	return setting.GetAuthorizer()
}
