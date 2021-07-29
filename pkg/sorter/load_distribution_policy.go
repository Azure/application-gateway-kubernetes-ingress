// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

// ByLoadDistributionPolicyName is a facility to sort slices of Kubernetes Ingress by their UID
type ByLoadDistributionPolicyName []n.ApplicationGatewayLoadDistributionPolicy ///appgw config type from go-sdk

func (a ByLoadDistributionPolicyName) Len() int      { return len(a) }
func (a ByLoadDistributionPolicyName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLoadDistributionPolicyName) Less(i, j int) bool {
	return getLoadDistributionPolicy(a[i]) < getLoadDistributionPolicy(a[j])
}

func getLoadDistributionPolicy(loadDistributionPolicy n.ApplicationGatewayLoadDistributionPolicy) string {
	return *loadDistributionPolicy.Name
}
