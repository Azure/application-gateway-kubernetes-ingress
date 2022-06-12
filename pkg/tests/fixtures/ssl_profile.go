// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	// CertificateName1 is a string constant.
	SslProfileName1 = "hardend-tls"
)

// GetSslProfile1 generates a certificate.
func GetSslProfile1() n.ApplicationGatewaySslProfile {
	return n.ApplicationGatewaySslProfile{
		Name: to.StringPtr(SslProfileName1),
	}
}
