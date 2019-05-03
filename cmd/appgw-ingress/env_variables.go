// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import "os"

type EnvVariables struct {
	SubscriptionID    string
	ResourceGroupName string
	AppGwName         string
	AuthLocation      string
	WatchNamespace    string
}

func NewEnvVariables() EnvVariables {
	return EnvVariables{
		SubscriptionID:    os.Getenv("APPGW_SUBSCRIPTION_ID"),
		ResourceGroupName: os.Getenv("APPGW_RESOURCE_GROUP"),
		AppGwName:         os.Getenv("APPGW_NAME"),
		AuthLocation:      os.Getenv("AZURE_AUTH_LOCATION"),
		WatchNamespace:    os.Getenv("KUBERNETES_WATCHNAMESPACE"),
	}
}
