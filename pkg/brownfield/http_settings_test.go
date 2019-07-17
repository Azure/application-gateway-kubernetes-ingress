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

var _ = Describe("Test blacklisting HTTP settings", func() {

	appGw := fixtures.GetAppGateway()

	sett1 := (*appGw.BackendHTTPSettingsCollection)[0]
	sett2 := (*appGw.BackendHTTPSettingsCollection)[1]
	sett3 := (*appGw.BackendHTTPSettingsCollection)[2]

	Context("Test GetBlacklistedHTTPSettings() with a blacklist", func() {
		It("should create a list of blacklisted and non blacklisted settings", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets() // Host: "bye.com", Paths: [/fox, /bar]
			er := NewExistingResources(appGw, prohibitedTargets, nil)

			blacklisted, nonBlacklisted := er.GetBlacklistedHTTPSettings()
			Expect(len(blacklisted)).To(Equal(2))
			Expect(len(nonBlacklisted)).To(Equal(1))

			Expect(blacklisted).To(ContainElement(sett1))
			Expect(blacklisted).To(ContainElement(sett2))

			Expect(nonBlacklisted).To(ContainElement(sett3))
		})
	})

	Context("Test GetBlacklistedHTTPSettings() with a blacklist with wild card", func() {
		It("should create a list of blacklisted and non blacklisted settings", func() {
			wildcard := &ptv1.AzureIngressProhibitedTarget{
				Spec: ptv1.AzureIngressProhibitedTargetSpec{},
			}
			prohibitedTargets := append(fixtures.GetAzureIngressProhibitedTargets(), wildcard)

			er := NewExistingResources(appGw, prohibitedTargets, nil)
			blacklisted, nonBlacklisted := er.GetBlacklistedHTTPSettings()
			Expect(len(blacklisted)).To(Equal(2))

			// One HTTP Setting is not associated with a listener, so we can't blacklist it.
			Expect(len(nonBlacklisted)).To(Equal(1))
		})
	})

	Context("Test getBlacklistedSettingsSet()", func() {
		It("should create a set of blacklisted settings", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets()
			er := NewExistingResources(appGw, prohibitedTargets, nil)
			set := er.getBlacklistedSettingsSet()
			Expect(len(set)).To(Equal(2))
			_, exists := set[fixtures.BackendHTTPSettingsName1]
			Expect(exists).To(BeTrue())
			_, exists = set[fixtures.BackendHTTPSettingsName2]
			Expect(exists).To(BeTrue())
		})
	})
})
