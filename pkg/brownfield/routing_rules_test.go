// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test blacklist request routing rules", func() {
	appGw := fixtures.GetAppGateway()

	ruleDefault := (*appGw.RequestRoutingRules)[0]
	ruleBasic := (*appGw.RequestRoutingRules)[1]
	rulePathBased1 := (*appGw.RequestRoutingRules)[2]
	rulePathBased2 := (*appGw.RequestRoutingRules)[3]

	Context("Test getRoutingRuleToTargetsMap()", func() {
		It("should create a map of routing rules to targets", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets() // Host: "bye.com", Paths: [/fox, /bar]
			er := NewExistingResources(appGw, prohibitedTargets, nil)

			ruleToTargets, pathMapToTargets := er.getRuleToTargets()

			Expect(len(ruleToTargets)).To(Equal(3))
			Expect(len(pathMapToTargets)).To(Equal(2))

			targetFoo := Target{Hostname: tests.Host, Path: fixtures.PathFoo}
			targetBar := Target{Hostname: tests.Host, Path: fixtures.PathBar}
			targetBaz := Target{Hostname: tests.Host, Path: fixtures.PathBaz}
			targetHostNoPath := Target{Hostname: tests.Host}

			Expect(len(ruleToTargets[fixtures.RequestRoutingRuleName1])).To(Equal(4))
			Expect(ruleToTargets[fixtures.RequestRoutingRuleName1]).To(ContainElement(targetFoo))
			Expect(ruleToTargets[fixtures.RequestRoutingRuleName1]).To(ContainElement(targetBar))
			Expect(ruleToTargets[fixtures.RequestRoutingRuleName1]).To(ContainElement(targetBaz))
			Expect(ruleToTargets[fixtures.RequestRoutingRuleName1]).To(ContainElement(targetHostNoPath))

			targetFox := Target{Hostname: tests.OtherHost, Path: fixtures.PathFox}
			targetOtherHostNoPath := Target{Hostname: tests.OtherHost}

			Expect(len(ruleToTargets[fixtures.RequestRoutingRuleName2])).To(Equal(3))
			Expect(ruleToTargets[fixtures.RequestRoutingRuleName2]).To(ContainElement(targetFox))
			Expect(ruleToTargets[fixtures.RequestRoutingRuleName2]).To(ContainElement(targetOtherHostNoPath))

		})
	})

	Context("Test GetBlacklistedRoutingRules() with a blacklist", func() {
		It("should create a list of blacklisted and non blacklisted request routing rules", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets() // Host: "bye.com", Paths: [/fox, /bar]
			er := NewExistingResources(appGw, prohibitedTargets, nil)
			blacklisted, nonBlacklisted := er.GetBlacklistedRoutingRules()

			Expect(len(blacklisted)).To(Equal(3))
			Expect(blacklisted).To(ContainElement(rulePathBased1))
			Expect(blacklisted).To(ContainElement(rulePathBased2))
			Expect(blacklisted).To(ContainElement(ruleBasic))

			Expect(len(nonBlacklisted)).To(Equal(1))
			Expect(nonBlacklisted).To(ContainElement(ruleDefault))

		})
	})

	Context("Test GetBlacklistedRoutingRules() with a blacklist with a wild card", func() {
		It("should create a list of blacklisted and non blacklisted request routing rules", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets() // Host: "bye.com", Paths: [/fox, /bar]
			wildcard := &ptv1.AzureIngressProhibitedTarget{}
			prohibitedTargets = append(prohibitedTargets, wildcard)
			er := NewExistingResources(appGw, prohibitedTargets, nil)

			blacklisted, nonBlacklisted := er.GetBlacklistedRoutingRules()

			Expect(len(blacklisted)).To(Equal(4))
			Expect(len(nonBlacklisted)).To(Equal(0))

			Expect(blacklisted).To(ContainElement(ruleBasic))
			Expect(blacklisted).To(ContainElement(rulePathBased1))
			Expect(blacklisted).To(ContainElement(rulePathBased2))
			Expect(blacklisted).To(ContainElement(ruleDefault))
		})
	})
})
