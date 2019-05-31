// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

// A facility to sort slices of ApplicationGatewayBackendAddress by IP, FQDN
type byIPFQDN []n.ApplicationGatewayBackendAddress

func (a byIPFQDN) Len() int      { return len(a) }
func (a byIPFQDN) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byIPFQDN) Less(i, j int) bool {
	return getIPFQDNKey(a[i]) < getIPFQDNKey(a[j])
}

func getIPFQDNKey(record n.ApplicationGatewayBackendAddress) string {
	fqdn := ""
	if record.Fqdn != nil {
		fqdn = *record.Fqdn
	}
	ip := ""
	if record.IPAddress != nil {
		ip = *record.IPAddress
	}
	return fmt.Sprintf("%s-%s", fqdn, ip)
}
