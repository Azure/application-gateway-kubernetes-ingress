// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test blacklist certificates", func() {

	appGw := fixtures.GetAppGateway()

	cert1 := (*appGw.SslCertificates)[0]
	cert2 := (*appGw.SslCertificates)[1]
	cert3 := (*appGw.SslCertificates)[2]

	Context("Test GetBlacklistedCertificates() with a blacklist", func() {
		It("should create a list of blacklisted and non blacklisted certificates", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets() // Host: "bye.com", Paths: [/fox, /bar]
			er := NewExistingResources(appGw, prohibitedTargets, nil)
			blacklisted, nonBlacklisted := er.GetBlacklistedCertificates()
			Expect(len(blacklisted)).To(Equal(3))
			Expect(len(nonBlacklisted)).To(Equal(0))

			Expect(blacklisted).To(ContainElement(cert1))
			Expect(blacklisted).To(ContainElement(cert2))
			Expect(blacklisted).To(ContainElement(cert3))
		})
	})

	Context("Test GetBlacklistedCertificates() with a blacklist with wild card", func() {
		It("should create a list of blacklisted and non blacklisted certificates", func() {
			wildcard := &ptv1.AzureIngressProhibitedTarget{
				Spec: ptv1.AzureIngressProhibitedTargetSpec{},
			}
			prohibitedTargets := append(fixtures.GetAzureIngressProhibitedTargets(), wildcard)

			er := NewExistingResources(appGw, prohibitedTargets, nil)
			blacklisted, nonBlacklisted := er.GetBlacklistedCertificates()
			Expect(len(blacklisted)).To(Equal(3))
			Expect(len(nonBlacklisted)).To(Equal(0))
		})
	})

	Context("Test getBlacklistedCertSet()", func() {
		It("should create a set of blacklisted certificates", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets()
			er := NewExistingResources(appGw, prohibitedTargets, nil)
			set := er.getBlacklistedCertSet()
			Expect(len(set)).To(Equal(3))
			_, exists := set[fixtures.CertificateName1]
			Expect(exists).To(BeTrue())
			_, exists = set[fixtures.CertificateName3]
			Expect(exists).To(BeTrue())
		})
	})

})
