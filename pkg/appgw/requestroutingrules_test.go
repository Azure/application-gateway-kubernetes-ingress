// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var expectedRedirectID = "/subscriptions/" + testFixturesSubscription +
	"/resourceGroups/" + testFixtureResourceGroup +
	"/providers/Microsoft.Network/applicationGateways/" + testFixtureAppGwName +
	"/redirectConfigurations/" + agPrefix + "sslr-" +
	testFixturesNamespace +
	"-" +
	testFixturesName

func makeHTTPURLPathMap() network.ApplicationGatewayURLPathMap {
	return network.ApplicationGatewayURLPathMap{
		Name: to.StringPtr("-path-map-name-"),
		ApplicationGatewayURLPathMapPropertiesFormat: &network.ApplicationGatewayURLPathMapPropertiesFormat{
			PathRules: &[]network.ApplicationGatewayPathRule{
				{
					ID:   to.StringPtr("-the-id-"),
					Type: to.StringPtr("-the-type-"),
					Etag: to.StringPtr("-the-etag-"),
					Name: to.StringPtr("/some/path"),
					ApplicationGatewayPathRulePropertiesFormat: &network.ApplicationGatewayPathRulePropertiesFormat{
						BackendAddressPool:    resourceRef("--BackendAddressPool--"),
						BackendHTTPSettings:   resourceRef("--BackendHTTPSettings--"),
						RedirectConfiguration: resourceRef("--RedirectConfiguration--"),
						RewriteRuleSet:        resourceRef("--RewriteRuleSet--"),
						ProvisioningState:     to.StringPtr("--provisionStateExpected--"),
					},
				},
			},
		},
	}
}

var _ = Describe("Test SSL Redirect Annotations", func() {
	Context("test getSslRedirectConfigResourceReference", func() {
		configBuilder := newConfigBuilderFixture(nil)
		ingress := newIngressFixture()

		actualID := configBuilder.getSslRedirectConfigResourceReference(ingress).ID

		It("generates expected ID", func() {
			Expect(expectedRedirectID).To(Equal(*actualID))
		})
	})

	Context("test modifyPathRulesForRedirection with 0 path rules", func() {
		configBuilder := newConfigBuilderFixture(nil)
		ingress := newIngressFixture()
		pathMap := makeHTTPURLPathMap()

		// Ensure there are no path rules defined for this test
		pathMap.PathRules = &[]network.ApplicationGatewayPathRule{}

		// Ensure the test is setup correctly
		It("should have 0 PathRules", func() {
			Expect(len(*pathMap.PathRules)).To(Equal(0))
		})

		// !! Action !! -- will mutate pathMap struct
		configBuilder.modifyPathRulesForRedirection(ingress, &pathMap)

		actualID := *(pathMap.DefaultRedirectConfiguration.ID)
		It("generated expected ID", func() {
			Expect(expectedRedirectID).To(Equal(actualID))
		})

		It("should still have 0 path rules", func() {
			Expect(0).To(Equal(len(*pathMap.PathRules)))
		})
	})

	Context("test modifyPathRulesForRedirection with 1 path rules", func() {
		configBuilder := newConfigBuilderFixture(nil)
		ingress := newIngressFixture()
		pathMap := makeHTTPURLPathMap()

		// Ensure the test is setup correctly
		It("should have length of PathRules to be 1", func() {
			Expect(1).To(Equal(len(*pathMap.PathRules)))
		})

		firstPathRule := (*pathMap.PathRules)[0]
		firstPathRule.BackendAddressPool = &network.SubResource{ID: to.StringPtr("-something-")}
		firstPathRule.BackendHTTPSettings = &network.SubResource{ID: to.StringPtr("-something-")}

		// !! Action !! -- will mutate pathMap struct
		configBuilder.modifyPathRulesForRedirection(ingress, &pathMap)

		actual := *(*pathMap.PathRules)[0].ApplicationGatewayPathRulePropertiesFormat

		It("should have a nil BackendAddressPool", func() {
			Expect(firstPathRule.BackendAddressPool).To(BeNil())
		})

		It("should have a nil BackendHTTPSettings", func() {
			Expect(firstPathRule.BackendHTTPSettings).To(BeNil())
		})

		It("should have correct RedirectConfiguration ID", func() {
			Expect(expectedRedirectID).To(Equal(*actual.RedirectConfiguration.ID))
		})
	})
})
