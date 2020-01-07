// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

// gettrustedRootCertificates obtains all SSL Certificates for the given Ingress object.
func (c *appGwConfigBuilder) getTrustedRootCertificates(cbCtx *ConfigBuilderContext) *[]n.ApplicationGatewayTrustedRootCertificate {
	if c.mem.trustedroots != nil {
		return c.mem.trustedroots
	}

	var trustedRootCertificates []n.ApplicationGatewayTrustedRootCertificate
	trustedRootCertificateMap := make(map[string]interface{}, 0)
	for _, ingress := range cbCtx.IngressList {
		trustedRootCertificateBase64, err := annotations.BackendTrustedRoot(ingress)
		if _, ok := trustedRootCertificateMap[trustedRootCertificateBase64]; !ok && err == nil {
			trustedRootCertificates = append(trustedRootCertificates, c.newTrustedRootCertificate(generateTrustedRootCertificateName(ingress), trustedRootCertificateBase64))
			trustedRootCertificateMap[trustedRootCertificateBase64] = nil
		}
	}

	sort.Sort(sorter.ByTrustedRootCertificateName(trustedRootCertificates))
	c.mem.trustedroots = &trustedRootCertificates
	return &trustedRootCertificates
}

func (c *appGwConfigBuilder) newTrustedRootCertificate(certName string, base64RootCert string) n.ApplicationGatewayTrustedRootCertificate {
	return n.ApplicationGatewayTrustedRootCertificate{
		Name: to.StringPtr(certName),
		ID:   to.StringPtr(c.appGwIdentifier.trustedRootCertificateID(certName)),
		ApplicationGatewayTrustedRootCertificatePropertiesFormat: &n.ApplicationGatewayTrustedRootCertificatePropertiesFormat{
			Data: to.StringPtr(base64RootCert),
		},
	}
}
