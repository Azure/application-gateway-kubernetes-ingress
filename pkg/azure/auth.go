// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/glog"
)

// WaitForAzureAuth waits until we can successfully get the gateway
func WaitForAzureAuth(azClient AzClient, maxAuthRetryCount int, retryPause time.Duration) error {
	retryCount := 0
	for {
		response, err := azClient.GetGateway()
		if err == nil {
			return nil
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
			return ErrAppGatewayNotFound
		}

		if response.Response.Response != nil && response.Response.StatusCode != 200 {
			// for example, getting 401. This is not expected as we are getting a token before making the call.
			glog.Error("Unexpected ARM status code on GET existing App Gateway config: ", response.Response.StatusCode)
		}

		if retryCount >= maxAuthRetryCount {
			glog.Errorf("Tried %d times to authenticate with ARM; Error: %s", retryCount, err)
			return ErrGetArmAuth
		}
		retryCount++
		glog.Errorf("Failed fetching config for App Gateway instance. Will retry in %v. Error: %s", retryPause, err)
		time.Sleep(retryPause)
	}
}

// GetAuthorizerWithRetry return azure.Authorizer
func GetAuthorizerWithRetry(authLocation string, maxAuthRetryCount int, retryPause time.Duration) (autorest.Authorizer, error) {
	var err error
	retryCount := 0
	for {
		// Fetch a new token
		if authorizer, err := getAuthorizer(authLocation); err == nil && authorizer != nil {
			return authorizer, nil
		}

		if retryCount >= maxAuthRetryCount {
			glog.Errorf("Tried %d times to get ARM authorization token; Error: %s", retryCount, err)
			return nil, ErrFailedGetToken
		}
		retryCount++
		glog.Errorf("Failed fetching authorization token for ARM. Will retry in %v. Error: %s", retryPause, err)
		time.Sleep(retryPause)
	}
}

func getAuthorizer(authLocation string) (autorest.Authorizer, error) {
	if authLocation == "" {
		// requires aad-pod-identity to be deployed in the AKS cluster
		// see https://github.com/Azure/aad-pod-identity for more information
		glog.V(1).Info("Creating authorizer from Azure Managed Service Identity")
		return auth.NewAuthorizerFromEnvironment()
	}
	glog.V(1).Infof("Creating authorizer from file referenced by environment variable: %s", authLocation)
	return auth.NewAuthorizerFromFile(n.DefaultBaseURI)
}
