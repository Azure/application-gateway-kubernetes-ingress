// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
)

// BySettingsName is a facility to sort slices of ApplicationGatewayBackendHTTPSettings by Name
type BySettingsName []n.ApplicationGatewayBackendHTTPSettings

func (a BySettingsName) Len() int      { return len(a) }
func (a BySettingsName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySettingsName) Less(i, j int) bool {
	return getSettingsName(a[i]) < getSettingsName(a[j])
}

func getSettingsName(setting n.ApplicationGatewayBackendHTTPSettings) string {
	if setting.Name == nil {
		return ""
	}
	return *setting.Name
}
