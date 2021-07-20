// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	"github.com/Azure/go-autorest/autorest/to"
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

const (
	// CertificateName1 is a string constant.
	CertificateName1 = "Certificate-1"

	// CertificateName2 is a string constant.
	CertificateName2 = "Certificate-2"

	// CertificateName3 is a string constant.
	CertificateName3 = "Certificate-3"

	// RootCertificateName1 is a string constant.
	RootCertificateName1 = "RootCertificate-1"

	// RootCertificateName2 is a string constant.
	RootCertificateName2 = "RootCertificate-2"

	// RootCertificateName3 is a string constant.
	RootCertificateName3 = "RootCertificate-3"
)

// GetCertificate1 generates a certificate.
func GetCertificate1() n.ApplicationGatewaySslCertificate {
	return n.ApplicationGatewaySslCertificate{
		Name: to.StringPtr(CertificateName1),
	}
}

// GetCertificate2 generates a certificate.
func GetCertificate2() n.ApplicationGatewaySslCertificate {
	return n.ApplicationGatewaySslCertificate{
		Name: to.StringPtr(CertificateName2),
	}
}

// GetCertificate3 generates a certificate.
func GetCertificate3() n.ApplicationGatewaySslCertificate {
	return n.ApplicationGatewaySslCertificate{
		Name: to.StringPtr(CertificateName3),
	}
}

// GetRootCertificate1 generates a root certificate.
func GetRootCertificate1() n.ApplicationGatewayTrustedRootCertificate {
	return n.ApplicationGatewayTrustedRootCertificate{
		Name: to.StringPtr(RootCertificateName1),
	}
}

// GetRootCertificate2 generates a root certificate.
func GetRootCertificate2() n.ApplicationGatewayTrustedRootCertificate {
	return n.ApplicationGatewayTrustedRootCertificate{
		Name: to.StringPtr(RootCertificateName2),
	}
}

// GetRootCertificate3 generates a root certificate.
func GetRootCertificate3() n.ApplicationGatewayTrustedRootCertificate {
	return n.ApplicationGatewayTrustedRootCertificate{
		Name: to.StringPtr(RootCertificateName3),
	}
}
