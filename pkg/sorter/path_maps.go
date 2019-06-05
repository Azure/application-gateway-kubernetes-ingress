// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

// ByPathMap is facility to sort slices of ApplicationGatewayURLPathMap by Name
type ByPathMap []n.ApplicationGatewayURLPathMap

func (a ByPathMap) Len() int      { return len(a) }
func (a ByPathMap) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByPathMap) Less(i, j int) bool {
	return getPathMapName(a[i]) < getPathMapName(a[j])
}

func getPathMapName(pathmap n.ApplicationGatewayURLPathMap) string {
	if pathmap.Name == nil {
		return ""
	}
	return *pathmap.Name
}
