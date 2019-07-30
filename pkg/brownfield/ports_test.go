// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test blacklisting ports", func() {

	appGw := fixtures.GetAppGateway()

	Context("Test getBlacklistedPortsSet()", func() {
		It("should create a set of blacklisted ports", func() {
			prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets()
			er := NewExistingResources(appGw, prohibitedTargets, nil)
			set := er.getBlacklistedPortsSet()
			Expect(len(set)).To(Equal(1))
		})
	})
})
