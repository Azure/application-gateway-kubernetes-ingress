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

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("test blacklist/whitelist health probes", func() {

	Context("test inProbeList", func() {
		{
			probe := fixtures.GetApplicationGatewayProbe(nil, nil)
			actual := inProbeList(&probe, getProhibitedTargetList(fixtures.GetProhibitedTargets()))
			It("should be able to find probe in prohibited Target list", func() {
				Expect(actual).To(BeFalse())
			})
		}
		{
			probe := fixtures.GetApplicationGatewayProbe(nil, nil)
			actual := inProbeList(&probe, getManagedTargetList(fixtures.GetManagedTargets()))
			It("should be able to find probe in managed Target list", func() {
				Expect(actual).To(BeTrue())
			})
		}
	})

	Context("test getManagedProbes", func() {

		whiteListedProbe := fixtures.GetApplicationGatewayProbe(nil, to.StringPtr("/baz")) // whitelisted
		probes := []n.ApplicationGatewayProbe{
			fixtures.GetApplicationGatewayProbe(nil, to.StringPtr("/fox")), // blacklisted
			fixtures.GetApplicationGatewayProbe(nil, to.StringPtr("/bar")), // blacklisted
			whiteListedProbe,
		}

		actual := GetManagedProbes(probes, fixtures.GetManagedTargets(), fixtures.GetProhibitedTargets())

		It("should have filtered probes based on black/white list", func() {
			Expect(len(actual)).To(Equal(1))
			Expect(actual).To(ContainElement(whiteListedProbe))
		})

	})
})
