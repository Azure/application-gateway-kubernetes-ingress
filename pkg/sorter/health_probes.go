// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

// ByHealthProbeName is a facility to sort slices of ApplicationGatewayProbe by Name
type ByHealthProbeName []n.ApplicationGatewayProbe

func (a ByHealthProbeName) Len() int      { return len(a) }
func (a ByHealthProbeName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByHealthProbeName) Less(i, j int) bool {
	return getHealthProbeName(a[i]) < getHealthProbeName(a[j])
}

func getHealthProbeName(probe n.ApplicationGatewayProbe) string {
	if probe.Name == nil {
		return ""
	}
	return *probe.Name
}
