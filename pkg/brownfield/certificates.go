// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

type certName string
type certsByName map[certName]n.ApplicationGatewaySslCertificate

// MergeCerts merges list of lists of certs into a single list, maintaining uniqueness.
func MergeCerts(certBuckets ...[]n.ApplicationGatewaySslCertificate) []n.ApplicationGatewaySslCertificate {
	uniq := make(certsByName)
	for _, bucket := range certBuckets {
		for _, cert := range bucket {
			uniq[certName(*cert.Name)] = cert
		}
	}
	merged := []n.ApplicationGatewaySslCertificate{}
	for _, cert := range uniq {
		merged = append(merged, cert)
	}
	return merged
}
