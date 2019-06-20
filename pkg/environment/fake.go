// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package environment

// GetFakeEnv returns fake values for defined environment variables for Ingress Controller.
func GetFakeEnv() EnvVariables {
	env := EnvVariables{
		SubscriptionID:    "--SubscriptionID--",
		ResourceGroupName: "--ResourceGroupName--",
		AppGwName:         "--AppGwName--",
		AuthLocation:      "--AuthLocation--",
		WatchNamespace:    "--WatchNamespace--",
		UsePrivateIP:      "false",
		VerbosityLevel:    "123456789",
	}

	return env
}
