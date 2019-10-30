// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
)

// ByListenerName is a facility to sort slices of ApplicationGatewayHTTPListener by Name
type ByListenerName []n.ApplicationGatewayHTTPListener

func (a ByListenerName) Len() int      { return len(a) }
func (a ByListenerName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByListenerName) Less(i, j int) bool {
	return getListenerName(a[i]) < getListenerName(a[j])
}

func getListenerName(listener n.ApplicationGatewayHTTPListener) string {
	if listener.Name == nil {
		return ""
	}
	return *listener.Name
}
