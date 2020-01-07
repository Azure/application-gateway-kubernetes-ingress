// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// WaitForAzureAuth waits until we can successfully get the gateway
func WaitForAzureAuth(azClient AzClient, maxAuthRetryCount int, retryPause time.Duration) error {
	err := utils.Retry(maxAuthRetryCount, retryPause,
		func() (bool, error) {
			response, err := azClient.GetGateway()
			if err == nil {
				return false, nil
			}

			// Reasons for 403 errors
			if response.Response.Response != nil && response.Response.StatusCode == 403 {
				glog.Error("Possible reasons:" +
					" AKS Service Principal requires 'Managed Identity Operator' access on Controller Identity;" +
					" 'identityResourceID' and/or 'identityClientID' are incorrect in the Helm config;" +
					" AGIC Identity requires 'Contributor' access on Application Gateway and 'Reader' access on Application Gateway's Resource Group;")
			}

			if response.Response.Response != nil && response.Response.StatusCode == 404 {
				glog.Error("Got 404 NOT FOUND status code on getting Application Gateway from ARM.")
				return false, ErrAppGatewayNotFound
			}

			if response.Response.Response != nil && response.Response.StatusCode != 200 {
				// for example, getting 401. This is not expected as we are getting a token before making the call.
				glog.Error("Unexpected ARM status code on GET existing App Gateway config: ", response.Response.StatusCode)
			}

			glog.Errorf("Failed fetching config for App Gateway instance. Will retry in %v. Error: %s", retryPause, err)
			return true, ErrGetArmAuth
		})

	if err != ErrAppGatewayNotFound {
		glog.Errorf("Tried %d times to authenticate with ARM; Error: %s", maxAuthRetryCount, err)
	}

	return err
}

// GetAuthorizerWithRetry return azure.Authorizer
func GetAuthorizerWithRetry(authLocation string, useManagedidentity bool, azContext *AzContext, maxAuthRetryCount int, retryPause time.Duration) (authorizer autorest.Authorizer, err error) {
	utils.Retry(maxAuthRetryCount, retryPause,
		func() (bool, error) {
			// Fetch a new token
			authorizer, err = getAuthorizer(authLocation, useManagedidentity, azContext)
			return true, err
		})
	return authorizer, nil
}

func getAuthorizer(authLocation string, useManagedidentity bool, azContext *AzContext) (autorest.Authorizer, error) {
	// Authorizer logic:
	// 1. If User provided authLocation, then use the file.
	// 2. If User provided a managed identity in ex: helm config, then use Environment
	// 3. If User provided nothing and AzContext has value, then use AzContext
	// 4. Fall back to environment
	if authLocation != "" {
		glog.V(1).Infof("Creating authorizer from file referenced by environment variable: %s", authLocation)
		return auth.NewAuthorizerFromFile(n.DefaultBaseURI)
	}
	if !useManagedidentity && azContext != nil {
		glog.V(1).Info("Creating authorizer using Cluster Service Principal.")
		credAuthorizer := auth.NewClientCredentialsConfig(azContext.ClientID, azContext.ClientSecret, azContext.TenantID)
		return credAuthorizer.Authorizer()
	}

	glog.V(1).Info("Creating authorizer from Azure Managed Service Identity")
	return auth.NewAuthorizerFromEnvironment()
}
