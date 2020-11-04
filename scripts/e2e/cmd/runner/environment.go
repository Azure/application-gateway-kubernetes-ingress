// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package runner

import (
	"fmt"
	"os"
	"regexp"

	"github.com/golang/glog"
)

const (
	// SubscriptionIDVarName is the name of the subscriptionId
	SubscriptionIDVarName = "subscriptionId"

	// ResourceGroupNameVarName is the name of the applicationGatewayResourceGroup
	ResourceGroupNameVarName = "applicationGatewayResourceGroup"

	// AppGwNameVarName is the name of the applicationGatewayName
	AppGwNameVarName = "applicationGatewayName"

	// KubeConfigVarName is the name of the KUBECONFIG
	KubeConfigVarName = "KUBECONFIG"

	// ObjectIDVarName is the name of the identityObjectId
	ObjectIDVarName = "identityObjectId"

	// AzureAuthLocationVarName is the name of the AZURE_AUTH_LOCATION
	AzureAuthLocationVarName = "AZURE_AUTH_LOCATION"

	// SubResourceNamePrefixVarName is the name of the subResourceNamePrefix
	SubResourceNamePrefixVarName = "subResourceNamePrefix"
)

// EnvVariables is a struct storing values for environment variables.
type EnvVariables struct {
	SubscriptionID        string
	ResourceGroupName     string
	AppGwName             string
	SubResourceNamePrefix string
	KubeConfigFilePath    string
	ObjectID              string
	AzureAuthLocation     string
}

// GetEnv returns values for defined environment variables for Ingress Controller.
func GetEnv() *EnvVariables {
	return &EnvVariables{
		SubscriptionID:        os.Getenv(SubscriptionIDVarName),
		ResourceGroupName:     os.Getenv(ResourceGroupNameVarName),
		AppGwName:             os.Getenv(AppGwNameVarName),
		SubResourceNamePrefix: os.Getenv(SubResourceNamePrefixVarName),
		KubeConfigFilePath:    GetEnvironmentVariable(KubeConfigVarName, "~/.kube/config", nil),
		ObjectID:              os.Getenv(ObjectIDVarName),
		AzureAuthLocation:     os.Getenv(AzureAuthLocationVarName),
	}
}

// GetResourceGroupID returns group's resource id
func (env *EnvVariables) GetResourceGroupID() string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", env.SubscriptionID, env.ResourceGroupName)
}

// GetApplicationGatewayResourceID returns gateway's resource id
func (env *EnvVariables) GetApplicationGatewayResourceID() string {
	return fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/applicationGateways/%s",
		env.SubscriptionID,
		env.ResourceGroupName,
		env.AppGwName)
}

// GetEnvironmentVariable is an augmentation of os.Getenv, providing it with a default value.
func GetEnvironmentVariable(environmentVariable, defaultValue string, validator *regexp.Regexp) string {
	if value, ok := os.LookupEnv(environmentVariable); ok {
		if validator == nil {
			return value
		}
		if validator.MatchString(value) {
			return value
		}
		glog.Errorf("Environment variable %s contains a value which does not pass validation filter; Using default value: %s", environmentVariable, defaultValue)
	}
	return defaultValue
}
