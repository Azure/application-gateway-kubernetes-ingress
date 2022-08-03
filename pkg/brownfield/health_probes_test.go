// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/mocks"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test blacklist health probes", func() {

	appGw := fixtures.GetAppGateway()

	managedProbe := (*appGw.Probes)[0]
	blacklistedProbe := (*appGw.Probes)[1]

	Context("Test GetBlacklistedProbes() with a blacklist", func() {

		It("should create a list of blacklisted and non blacklisted health probes", func() {

			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets() // /fox  /bar

			er := NewExistingResources(appGw, prohibitedTargets, nil)

			blacklisted, nonBlacklisted := er.GetBlacklistedProbes()

			// When there is both blacklist and whitelist - the whitelist is ignored.
			Expect(len(blacklisted)).To(Equal(2))
			Expect(len(nonBlacklisted)).To(Equal(1))

			// not explicitly blacklisted (does not matter that it is in blacklist)
			Expect(blacklisted).To(ContainElement(managedProbe))
			Expect(blacklisted).To(ContainElement(blacklistedProbe))
		})
	})

	Context("Test GetBlacklistedProbes() with a blacklist containing a wildcard blacklist target", func() {

		It("should create a list of blacklisted and non blacklisted health probes with a wildcard", func() {

			wildcard := &ptv1.AzureIngressProhibitedTarget{
				Spec: ptv1.AzureIngressProhibitedTargetSpec{},
			}
			prohibitedTargets := append(fixtures.GetAzureIngressProhibitedTargets(), wildcard)

			er := NewExistingResources(appGw, prohibitedTargets, nil)

			// Everything is blacklisted
			blacklisted, nonBlacklisted := er.GetBlacklistedProbes()

			Expect(len(blacklisted)).To(Equal(2))
			Expect(len(nonBlacklisted)).To(Equal(1))

			Expect(blacklisted).To(ContainElement(managedProbe))
			Expect(blacklisted).To(ContainElement(blacklistedProbe))
		})
	})

	Context("Test MergeProbes()", func() {

		probeList1 := []n.ApplicationGatewayProbe{
			managedProbe,
		}

		probeList2 := []n.ApplicationGatewayProbe{
			managedProbe,
			blacklistedProbe,
		}

		probeList3 := []n.ApplicationGatewayProbe{
			blacklistedProbe,
		}

		It("should correctly merge lists of probes", func() {
			merge1 := MergeProbes(probeList2, probeList3)
			Expect(len(merge1)).To(Equal(2))
			Expect(merge1).To(ContainElement(managedProbe))
			Expect(merge1).To(ContainElement(blacklistedProbe))

			merge2 := MergeProbes(probeList1, probeList3)
			Expect(len(merge2)).To(Equal(2))
			Expect(merge1).To(ContainElement(managedProbe))
			Expect(merge1).To(ContainElement(blacklistedProbe))
		})
	})

	Context("Test getBlacklistedProbesSet()", func() {
		It("should create a set of blacklisted probes", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets()
			er := NewExistingResources(appGw, prohibitedTargets, nil)
			set := er.getBlacklistedProbesSet()
			Expect(len(set)).To(Equal(2))
			_, exists := set[fixtures.ProbeName1]
			Expect(exists).To(BeTrue())
			_, exists = set[fixtures.ProbeName2]
			Expect(exists).To(BeTrue())
		})
	})

	Context("Test LogHTTPSettings()", func() {
		It("should log settings", func() {
			probes := []n.ApplicationGatewayProbe{
				fixtures.GetApplicationGatewayProbe(to.StringPtr("x"), to.StringPtr("y")),
			}
			logger := &mocks.MockLogger{}

			LogProbes(logger, probes, probes, probes)

			expected1 := "[brownfield] Probes AGIC created: _probe-name-eA-eQ"
			Expect(logger.LogLines).To(ContainElement(expected1))

			expected2 := "[brownfield] Existing Blacklisted Probes AGIC will retain: _probe-name-eA-eQ"
			Expect(logger.LogLines).To(ContainElement(expected2))
		})
	})
})
