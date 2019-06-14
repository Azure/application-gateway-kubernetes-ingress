// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"os"

	"github.com/golang/glog"
)

type envVariables struct {
	SubscriptionID    string
	ResourceGroupName string
	AppGwName         string
	AuthLocation      string
	WatchNamespace    string
}

func getEnvVars() envVariables {
	env := envVariables{
		SubscriptionID:    os.Getenv("APPGW_SUBSCRIPTION_ID"),
		ResourceGroupName: os.Getenv("APPGW_RESOURCE_GROUP"),
		AppGwName:         os.Getenv("APPGW_NAME"),
		AuthLocation:      os.Getenv("AZURE_AUTH_LOCATION"),
		WatchNamespace:    os.Getenv("KUBERNETES_WATCHNAMESPACE"),
	}

	if len(env.SubscriptionID) == 0 || len(env.ResourceGroupName) == 0 || len(env.AppGwName) == 0 {
		glog.Fatalf("Error while initializing values from environment. Please check helm configuration for missing values.")
	}

	if env.WatchNamespace == "" {
		glog.Info("KUBERNETES_WATCHNAMESPACE is not set. Watching all available namespaces.")
	}

	return env
}
