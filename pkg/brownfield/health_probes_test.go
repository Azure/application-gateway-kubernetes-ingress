// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test blacklist health probes", func() {

	managedProbe := fixtures.GetApplicationGatewayProbe(nil, to.StringPtr(fixtures.PathFoo))
	blacklistedProbe := fixtures.GetApplicationGatewayProbe(nil, to.StringPtr(fixtures.PathBar))
	blacklistedByHost := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.OtherHost), nil)

	managedProbeWeirdURL := fixtures.GetApplicationGatewayProbe(nil, to.StringPtr(fixtures.PathFoo+"/healthz"))
	blackListedProbeWeirdURL := fixtures.GetApplicationGatewayProbe(nil, to.StringPtr(fixtures.PathBar+"/healthz"))
	blacklistedByHostWeirdURL := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.OtherHost), to.StringPtr("/healthz"))

	probes := []n.ApplicationGatewayProbe{
		managedProbe,     // /foo
		blacklistedProbe, // /bar
		blacklistedByHost,

		managedProbeWeirdURL,      // /foo/healthz
		blackListedProbeWeirdURL,  // /bar/healthz
		blacklistedByHostWeirdURL, // /healthz
	}

	Context("Test GetBlacklistedProbes() with a blacklist", func() {

		It("should create a list of blacklisted and non blacklisted health probes", func() {

			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets() // /fox  /bar

			blacklisted, nonBlacklisted := GetBlacklistedProbes(probes, prohibitedTargets)

			// When there is both blacklist and whitelist - the whitelist is ignored.
			Expect(len(blacklisted)).To(Equal(4))

			// not explicitly blacklisted (does not matter that it is in blacklist)
			Expect(blacklisted).ToNot(ContainElement(managedProbe))
			Expect(blacklisted).ToNot(ContainElement(managedProbeWeirdURL))

			// not explicitly blacklisted -- it is in neither blacklist or whitelist
			Expect(blacklisted).To(ContainElement(blacklistedByHost))
			Expect(blacklisted).To(ContainElement(blacklistedByHostWeirdURL))

			// explicitly blacklisted
			Expect(blacklisted).To(ContainElement(blacklistedProbe))
			Expect(blacklisted).To(ContainElement(blackListedProbeWeirdURL))

			Expect(nonBlacklisted).ToNot(ContainElement(blackListedProbeWeirdURL))
		})
	})

	Context("Test GetBlacklistedProbes() with a blacklist containing a wildcard blacklist target", func() {

		It("should create a list of blacklisted and non blacklisted health probes with a wildcard", func() {

			wildcard := &ptv1.AzureIngressProhibitedTarget{
				Spec: ptv1.AzureIngressProhibitedTargetSpec{},
			}
			prohibitedTargets := append(fixtures.GetAzureIngressProhibitedTargets(), wildcard)

			// Everything is blacklisted
			blacklisted, nonBlacklisted := GetBlacklistedProbes(probes, prohibitedTargets)

			Expect(len(blacklisted)).To(Equal(6))
			Expect(len(nonBlacklisted)).To(Equal(0))

			Expect(blacklisted).To(ContainElement(managedProbe))
			Expect(blacklisted).To(ContainElement(managedProbeWeirdURL))

			// This is the only blacklisted probe
			Expect(blacklisted).To(ContainElement(blacklistedProbe))
			Expect(blacklisted).To(ContainElement(blackListedProbeWeirdURL))

			Expect(blacklisted).To(ContainElement(blacklistedByHost))
			Expect(blacklisted).To(ContainElement(blacklistedByHostWeirdURL))
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

	Context("test inProbeBlacklist with non blacklisted probe", func() {
		probe := fixtures.GetApplicationGatewayProbe(nil, nil)
		blacklist := GetTargetBlacklist(fixtures.GetAzureIngressProhibitedTargets())
		actual := inProbeBlacklist(&probe, blacklist)
		It("should be able to find probe in prohibited Target list", func() {
			Expect(actual).To(BeFalse())
		})
	})

	Context("test inProbeBlacklist with a blacklisted probe by Host only", func() {
		blacklist := GetTargetBlacklist(fixtures.GetAzureIngressProhibitedTargets())
		actual := inProbeBlacklist(&blacklistedByHost, blacklist)
		It("should be able to find probe in prohibited Target list", func() {
			Expect(actual).To(BeTrue())
		})
	})

	Context("test inProbeBlacklist with a blacklisted probe by Host and URL", func() {
		blacklist := GetTargetBlacklist(fixtures.GetAzureIngressProhibitedTargets())
		actual := inProbeBlacklist(&blacklistedByHostWeirdURL, blacklist)
		It("should be able to find probe in prohibited Target list", func() {
			Expect(actual).To(BeTrue())
		})
	})

	Context("test inProbeBlacklist with health probe to a sub-path", func() {
		target := Target{
			Hostname: tests.Host,
			Path:     "/abc/*",
		}
		targetList := []Target{target}

		It("should be able to match a probe in the sub-path of the target", func() {
			probe := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.Host), to.StringPtr("/abc/healthz"))
			actual := inProbeBlacklist(&probe, &targetList)
			Expect(actual).To(BeTrue())
		})

		It("should be able to find probe exactly matching the path of the target", func() {
			probe := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.Host), to.StringPtr("/abc"))

			actual := inProbeBlacklist(&probe, &targetList)
			Expect(actual).To(BeTrue())
		})

		It("should not be able to find probe not matching the target", func() {
			probe := fixtures.GetApplicationGatewayProbe(to.StringPtr(tests.Host), to.StringPtr("/xyz"))
			actual := inProbeBlacklist(&probe, &targetList)
			Expect(actual).To(BeFalse())
		})

	})
})
