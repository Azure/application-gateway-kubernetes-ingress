// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

// ByRequestRoutingRuleName is a facility to sort slices of ApplicationGatewayRequestRoutingRule by Name
type ByRequestRoutingRuleName []n.ApplicationGatewayRequestRoutingRule

func (a ByRequestRoutingRuleName) Len() int      { return len(a) }
func (a ByRequestRoutingRuleName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRequestRoutingRuleName) Less(i, j int) bool {
	return getRuleName(a[i]) < getRuleName(a[j])
}

func getRuleName(rule n.ApplicationGatewayRequestRoutingRule) string {
	if rule.Name == nil {
		return ""
	}
	return *rule.Name
}
