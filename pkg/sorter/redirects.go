// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
)

// ByRedirectName is a facility to sort slices of ApplicationGatewayRedirectConfiguration by Name
type ByRedirectName []n.ApplicationGatewayRedirectConfiguration

func (a ByRedirectName) Len() int      { return len(a) }
func (a ByRedirectName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRedirectName) Less(i, j int) bool {
	return getRedirectName(a[i]) < getRedirectName(a[j])
}

func getRedirectName(redirect n.ApplicationGatewayRedirectConfiguration) string {
	if redirect.Name == nil {
		return ""
	}
	return *redirect.Name
}
