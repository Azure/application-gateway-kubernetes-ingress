// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	// CertificateName1 is a string constant.
	CertificateName1 = "Certificate-1"

	// CertificateName2 is a string constant.
	CertificateName2 = "Certificate-2"

	// CertificateName3 is a string constant.
	CertificateName3 = "Certificate-3"
)

func GetCertificate1() n.ApplicationGatewaySslCertificate {
	return n.ApplicationGatewaySslCertificate{
		Name: to.StringPtr(CertificateName1),
	}
}

func GetCertificate2() n.ApplicationGatewaySslCertificate {
	return n.ApplicationGatewaySslCertificate{
		Name: to.StringPtr(CertificateName2),
	}
}

func GetCertificate3() n.ApplicationGatewaySslCertificate {
	return n.ApplicationGatewaySslCertificate{
		Name: to.StringPtr(CertificateName3),
	}
}
