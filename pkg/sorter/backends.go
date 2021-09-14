// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

// ByIPFQDN is a facility to sort slices of ApplicationGatewayBackendAddress by IP, FQDN
type ByIPFQDN []n.ApplicationGatewayBackendAddress

func (a ByIPFQDN) Len() int      { return len(a) }
func (a ByIPFQDN) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByIPFQDN) Less(i, j int) bool {
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
