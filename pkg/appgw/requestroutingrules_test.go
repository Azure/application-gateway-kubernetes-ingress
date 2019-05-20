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

const provisionStateExpected = "--provisionStateExpected--"
const rewriteRulesetID = "--RewriteRuleSet--"

var redirectConfigID = "/subscriptions/" + testFixturesSubscription +
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
						RewriteRuleSet:        resourceRef(rewriteRulesetID),
						ProvisioningState:     to.StringPtr(provisionStateExpected),
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
		actualID := *(configBuilder.getSslRedirectConfigResourceReference(ingress).ID)
		It("generates expected ID", func() {
			Expect(redirectConfigID).To(Equal(actualID))
		})
	})

	Context("test getSslRedirectConfigResourceReference", func() {
		configBuilder := newConfigBuilderFixture(nil)
		ingress := newIngressFixture()
		actualURLPathMap := makeHTTPURLPathMap()
		// Ensure there are no path rules defined for this test
		actualURLPathMap.PathRules = &[]network.ApplicationGatewayPathRule{}

		// Action -- will mutate actualURLPathMap struct
		configBuilder.addPathRules(ingress, &actualURLPathMap)

		actualID := *(actualURLPathMap.DefaultRedirectConfiguration.ID)
		It("generated expected ID", func() {
			Expect(redirectConfigID).To(Equal(actualID))
		})

		It("should still have 0 path rules", func() {
			Expect(0).To(Equal(len(*actualURLPathMap.PathRules)))
		})
	})

	Context("test getSslRedirectConfigResourceReference", func() {
		configBuilder := newConfigBuilderFixture(nil)
		ingress := newIngressFixture()
		pathMap := makeHTTPURLPathMap()

		// Ensure the test is setup correctly
		It("should have length of PathRules to be 1", func() {
			Expect(1).To(Equal(len(*pathMap.PathRules)))
		})

		// Action -- will mutate pathMap struct
		configBuilder.addPathRules(ingress, &pathMap)

		// Ensure the test is setup correctly
		actual := *(*pathMap.PathRules)[0].ApplicationGatewayPathRulePropertiesFormat

		It("sohuld have a nil BackendAddressPool", func() {
			Expect(actual.BackendAddressPool).To(BeNil())
		})

		It("should have a nil BackendHTTPSettings", func() {
			Expect(actual.BackendHTTPSettings).To(BeNil())
		})

		It("sohuld have correct RedirectConfiguration.ID", func() {
			Expect(redirectConfigID).To(Equal(*actual.RedirectConfiguration.ID))
		})

		It("should have correct RewriteRuleSet.ID", func() {
			Expect(rewriteRulesetID).To(Equal(*actual.RewriteRuleSet.ID))
		})

		It("should have correct ProvisioningState", func() {
			Expect(provisionStateExpected).To(Equal(*actual.ProvisioningState))
		})
	})
})
