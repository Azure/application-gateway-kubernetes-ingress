// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

var _ = Describe("Test SSL Redirect Annotations", func() {
	targetListener := listenerIdentifier{
		FrontendPort: int32(443),
		HostName:     "foo.baz",
	}

	expectedRedirectID := "/subscriptions/--subscription--" +
		"/resourceGroups/--resource-group--" +
		"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
		"/redirectConfigurations/sslr-fl-foo.baz-443"

	// TODO(draychev): Move to test fixtures
	ingress := fixtures.GetIngress()

	ingressList := []*v1beta1.Ingress{ingress}
	serviceList := []*v1.Service{tests.NewServiceFixture()}
	cbCtx := &ConfigBuilderContext{
		IngressList: ingressList,
		ServiceList: serviceList,
	}

	Context("test getSslRedirectConfigResourceReference", func() {
		configBuilder := newConfigBuilderFixture(nil)
		_ = configBuilder.k8sContext.Caches.Service.Add(tests.NewServiceFixture())

		actualID := configBuilder.getSslRedirectConfigResourceReference(targetListener).ID

		It("generates expected ID", func() {
			Expect(*actualID).To(Equal(expectedRedirectID))
		})
	})

	Context("test modifyPathRulesForRedirection with 0 path rules", func() {
		configBuilder := newConfigBuilderFixture(nil)
		pathMap := newURLPathMap()

		// Ensure there are no path rules defined for this test
		pathMap.PathRules = &[]n.ApplicationGatewayPathRule{}

		// Ensure the test is setup correctly
		It("should have 0 PathRules", func() {
			Expect(len(*pathMap.PathRules)).To(Equal(0))
		})

		// !! Action !! -- will mutate pathMap struct
		configBuilder.modifyPathRulesForRedirection(cbCtx, &pathMap, targetListener)

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
		pathMap := newURLPathMap()

		// Ensure the test is setup correctly
		It("should have length of PathRules to be 1", func() {
			Expect(1).To(Equal(len(*pathMap.PathRules)))
		})

		firstPathRule := (*pathMap.PathRules)[0]
		firstPathRule.BackendAddressPool = &n.SubResource{ID: to.StringPtr("-something-")}
		firstPathRule.BackendHTTPSettings = &n.SubResource{ID: to.StringPtr("-something-")}
		firstPathRule.RedirectConfiguration = nil

		// !! Action !! -- will mutate pathMap struct
		configBuilder.modifyPathRulesForRedirection(cbCtx, &pathMap, targetListener)

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

	Context("test RequestRoutingRules without HTTPS but with SSL Redirect", func() {
		configBuilder := newConfigBuilderFixture(nil)
		_ = configBuilder.k8sContext.Caches.Service.Add(tests.NewServiceFixture())
		_ = configBuilder.Listeners(cbCtx)
		_ = configBuilder.RequestRoutingRules(cbCtx)

		It("should have correct RequestRoutingRules", func() {
			Expect(len(*configBuilder.appGw.RequestRoutingRules)).To(Equal(2))

			Expect(*configBuilder.appGw.RequestRoutingRules).To(ContainElement(n.ApplicationGatewayRequestRoutingRule{
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType:            "Basic",
					BackendAddressPool:  nil,
					BackendHTTPSettings: nil,
					HTTPListener: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-foo.baz-80"),
					},
					URLPathMap:     nil,
					RewriteRuleSet: nil,
					RedirectConfiguration: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
							"/redirectConfigurations/sslr-fl-foo.baz-443")},
					ProvisioningState: nil,
				},
				Name: to.StringPtr("rr-foo.baz-80"),
				Etag: to.StringPtr("*"),
				Type: nil,
				ID:   to.StringPtr(configBuilder.appGwIdentifier.requestRoutingRuleID("rr-foo.baz-80")),
			}))

			Expect(*configBuilder.appGw.RequestRoutingRules).To(ContainElement(n.ApplicationGatewayRequestRoutingRule{
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType: "Basic",
					BackendAddressPool: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
							"/backendAddressPools/defaultaddresspool"),
					},
					BackendHTTPSettings: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
							"/backendHttpSettingsCollection/defaulthttpsetting"),
					},
					HTTPListener: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-foo.baz-443"),
					},
					URLPathMap:            nil,
					RewriteRuleSet:        nil,
					RedirectConfiguration: nil,
					ProvisioningState:     nil,
				},
				Name: to.StringPtr("rr-foo.baz-443"),
				Etag: to.StringPtr("*"),
				Type: nil,
				ID:   to.StringPtr(configBuilder.appGwIdentifier.requestRoutingRuleID("rr-foo.baz-443")),
			}))
		})

		It("should have correct URLPathMaps", func() {
			Expect(len(*configBuilder.appGw.URLPathMaps)).To(Equal(0))
		})
	})
})
