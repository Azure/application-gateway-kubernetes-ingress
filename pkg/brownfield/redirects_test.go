// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test GetBlacklistedRedirects", func() {

	prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets()

	redirects := []n.ApplicationGatewayRedirectConfiguration{
		{
			Name: to.StringPtr("redirect-1"),
			ApplicationGatewayRedirectConfigurationPropertiesFormat: &n.ApplicationGatewayRedirectConfigurationPropertiesFormat{},
		},
	}
	appGw := fixtures.GetAppGateway()

	er := NewExistingResources(appGw, prohibitedTargets, nil)

	Context("Test GetBlacklistedRedirects()", func() {
		It("should work as expected", func() {
			blacklistedRedirects, nonBlacklistedRedirects := er.GetBlacklistedRedirects()
			Expect(blacklistedRedirects).To(BeEmpty())
			Expect(nonBlacklistedRedirects).To(ContainElement(redirects[0]))
		})
	})

	Context("Test getBlacklistedRedirectsSet()", func() {
		It("should work as expected", func() {
			blacklistedRedirects := er.getBlacklistedRedirectsSet()
			expected := map[redirectName]interface{}{
				"RedirectConfiguration-2": nil,
				"RedirectConfiguration-1": nil,
				"":                        nil,
			}
			Expect(blacklistedRedirects).To(Equal(expected))
		})
	})

	Context("Test indexRedirectsByName()", func() {
		It("should create a set of the index names", func() {
			actual := indexRedirectsByName(redirects)
			expected := redirectsByName{
				"redirect-1": redirects[0],
			}
			Expect(actual).To(Equal(expected))
		})
	})

})
