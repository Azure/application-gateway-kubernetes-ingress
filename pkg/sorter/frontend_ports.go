// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
)

// ByFrontendPortName is a facility to sort slices of ApplicationGatewayFrontendPort by Name
type ByFrontendPortName []n.ApplicationGatewayFrontendPort

func (a ByFrontendPortName) Len() int      { return len(a) }
func (a ByFrontendPortName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByFrontendPortName) Less(i, j int) bool {
	return getFrontendPortName(a[i]) < getFrontendPortName(a[j])
}

func getFrontendPortName(port n.ApplicationGatewayFrontendPort) string {
	if port.Name == nil {
		return ""
	}
	return *port.Name
}
