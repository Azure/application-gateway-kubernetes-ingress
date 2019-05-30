// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

// A facility to sort slices of ApplicationGatewayBackendAddress by IP, FQDN
type byIPFQDN []n.ApplicationGatewayBackendAddress

func (a byIPFQDN) Len() int      { return len(a) }
func (a byIPFQDN) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byIPFQDN) Less(i, j int) bool {
	if a[i].IPAddress != nil && a[j].IPAddress != nil && len(*a[i].IPAddress) > 0 {
		return *a[i].IPAddress < *a[j].IPAddress
	} else if a[i].Fqdn != nil && a[j].Fqdn != nil && len(*a[i].Fqdn) != 0 {
		return *a[i].Fqdn < *a[j].Fqdn
	}
	return false
}
