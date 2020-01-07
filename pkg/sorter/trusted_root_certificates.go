// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package sorter

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
)

// ByTrustedRootCertificateName is a facility to sort slices of ApplicationGatewayTrustedRootCertificate by Name
type ByTrustedRootCertificateName []n.ApplicationGatewayTrustedRootCertificate

func (a ByTrustedRootCertificateName) Len() int      { return len(a) }
func (a ByTrustedRootCertificateName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTrustedRootCertificateName) Less(i, j int) bool {
	return getTrustedRootCertificate(a[i]) < getTrustedRootCertificate(a[j])
}

func getTrustedRootCertificate(cert n.ApplicationGatewayTrustedRootCertificate) string {
	if cert.Name == nil {
		return ""
	}
	return *cert.Name
}
