// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

// ByBackendPoolName is a facility to sort slices of ApplicationGatewayBackendAddressPool by Name
type ByBackendPoolName []n.ApplicationGatewayBackendAddressPool

func (a ByBackendPoolName) Len() int      { return len(a) }
func (a ByBackendPoolName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByBackendPoolName) Less(i, j int) bool {
	return getPoolName(a[i]) < getPoolName(a[j])
}

func getPoolName(redirect n.ApplicationGatewayBackendAddressPool) string {
	if redirect.Name == nil {
		return ""
	}
	return *redirect.Name
}
