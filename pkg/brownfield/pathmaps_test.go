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

var _ = Describe("Test blacklisting path maps", func() {

	appGw := fixtures.GetAppGateway()

	pathMap1 := (*appGw.URLPathMaps)[0]
	pathMap2 := (*appGw.URLPathMaps)[1]
	pathMap3 := (*appGw.URLPathMaps)[2]
	pathMap4 := (*appGw.URLPathMaps)[3]

	Context("Test GetBlacklistedHTTPSettings() with a blacklist", func() {
		It("should create a list of blacklisted and non blacklisted path maps", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets()
			er := NewExistingResources(appGw, prohibitedTargets, nil)

			blacklisted, nonBlacklisted := er.GetBlacklistedPathMaps()
			Expect(len(blacklisted)).To(Equal(3))
			Expect(blacklisted).To(ContainElement(pathMap2))
			Expect(blacklisted).To(ContainElement(pathMap3))
			Expect(blacklisted).To(ContainElement(pathMap4))

			Expect(len(nonBlacklisted)).To(Equal(1))
			Expect(nonBlacklisted).To(ContainElement(pathMap1))
		})
	})

	Context("Test GetBlacklistedHTTPSettings() with a blacklist with a wild card", func() {
		It("should create a list of blacklisted and non blacklisted path maps", func() {
			wildcard := &ptv1.AzureIngressProhibitedTarget{
				Spec: ptv1.AzureIngressProhibitedTargetSpec{},
			}
			prohibitedTargets := append(fixtures.GetAzureIngressProhibitedTargets(), wildcard)

			er := NewExistingResources(appGw, prohibitedTargets, nil)
			blacklisted, nonBlacklisted := er.GetBlacklistedPathMaps()
			Expect(len(blacklisted)).To(Equal(3))
			Expect(blacklisted).To(ContainElement(pathMap2))
			Expect(blacklisted).To(ContainElement(pathMap3))
			Expect(blacklisted).To(ContainElement(pathMap4))

			// One HTTP Setting is not associated with a listener, so we can't blacklist it.
			Expect(len(nonBlacklisted)).To(Equal(1))
			Expect(nonBlacklisted).To(ContainElement(pathMap1))
		})
	})
})
