// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/mocks"
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

	Context("Test MergeHTTPSettings()", func() {
		It("should merge buckets of settings", func() {
			sett1 := []n.ApplicationGatewayBackendHTTPSettings{
				fixtures.GetHTTPSettings1(),
				fixtures.GetHTTPSettings2(),
			}
			sett2 := []n.ApplicationGatewayBackendHTTPSettings{
				fixtures.GetHTTPSettings1(),
				fixtures.GetHTTPSettings3(),
			}
			actual := MergeHTTPSettings(sett1, sett2)
			Expect(actual).To(ContainElement(fixtures.GetHTTPSettings1()))
			Expect(actual).To(ContainElement(fixtures.GetHTTPSettings2()))
			Expect(actual).To(ContainElement(fixtures.GetHTTPSettings3()))
		})
	})

	Context("Test LogHTTPSettings()", func() {
		It("should log settings", func() {
			sett1 := []n.ApplicationGatewayBackendHTTPSettings{
				fixtures.GetHTTPSettings1(),
				fixtures.GetHTTPSettings2(),
			}
			sett2 := []n.ApplicationGatewayBackendHTTPSettings{
				fixtures.GetHTTPSettings1(),
				fixtures.GetHTTPSettings3(),
			}
			logger := &mocks.MockLogger{}

			LogHTTPSettings(logger, sett1, sett2, sett2)

			expected1 := "[brownfield] Existing Blacklisted HTTP Settings AGIC will retain:" +
				" _BackendHTTPSettings-1, BackendHTTPSettings-2"
			Expect(logger.LogLines).To(ContainElement(expected1))

			expected2 := "[brownfield] HTTP Settings AGIC created: _BackendHTTPSettings-1, BackendHTTPSettings-3"
			Expect(logger.LogLines).To(ContainElement(expected2))
		})
	})
})
