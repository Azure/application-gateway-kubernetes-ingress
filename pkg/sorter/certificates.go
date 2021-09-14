// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

// ByCertificateName is a facility to sort slices of ApplicationGatewaySslCertificate by Name
type ByCertificateName []n.ApplicationGatewaySslCertificate

func (a ByCertificateName) Len() int      { return len(a) }
func (a ByCertificateName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByCertificateName) Less(i, j int) bool {
	return getCertificateName(a[i]) < getCertificateName(a[j])
}

func getCertificateName(cert n.ApplicationGatewaySslCertificate) string {
	if cert.Name == nil {
		return ""
	}
	return *cert.Name
}
