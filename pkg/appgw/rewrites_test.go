// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayrewrite/v1beta1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/go-autorest/autorest/to"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Test the creation of Rewrite Rule Sets from Ingress definition", func() {
	defer GinkgoRecover()

	Context("1 ingress without rewrite rule set", func() {
		ing := tests.NewIngressFixture()
		ingressList := []*networking.Ingress{
			ing,
		}

		cb := newConfigBuilderFixture(nil)
		for _, ingress := range ingressList {
			_ = cb.k8sContext.Caches.Ingress.Add(ingress)
		}

		serviceList := []*v1.Service{
			tests.NewServiceFixture(),
		}

		cbCtx := &ConfigBuilderContext{
			IngressList:           ingressList,
			ServiceList:           serviceList,
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = cb.RewriteRuleSets(cbCtx)

		It("should contain correct number of rewrite rule sets", func() {
			Expect(len(*cb.appGw.RewriteRuleSets)).To(Equal(0))
		})
	})

	Context("1 ingress with rewrite rule set", func() {
		ing := tests.NewIngressFixture()
		ing.Annotations[annotations.RewriteRuleSetCustomResourceKey] = tests.RewriteRuleSetName

		ingressList := []*networking.Ingress{
			ing,
		}

		cb := newConfigBuilderFixture(nil)
		for _, ingress := range ingressList {
			_ = cb.k8sContext.Caches.Ingress.Add(ingress)
		}

		rewriteRuleSet := tests.NewRewriteRuleSetCustomResourceFixture(tests.RewriteRuleSetName)
		cb.k8sContext.Caches.AzureApplicationGatewayRewrite.Add(rewriteRuleSet)

		serviceList := []*v1.Service{
			tests.NewServiceFixture(),
		}

		cbCtx := &ConfigBuilderContext{
			IngressList:           ingressList,
			ServiceList:           serviceList,
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = cb.RewriteRuleSets(cbCtx)

		It("should contain correct number of rewrite rule sets", func() {
			Expect(len(*cb.appGw.RewriteRuleSets)).To(Equal(1))
		})
	})

	Context("2 ingress without rewrite rule set", func() {
		ing1 := tests.NewIngressFixture()
		ing2 := tests.NewIngressFixture()
		ingressList := []*networking.Ingress{
			ing1,
			ing2,
		}
		cb := newConfigBuilderFixture(nil)
		for _, ingress := range ingressList {
			_ = cb.k8sContext.Caches.Ingress.Add(ingress)
		}
		serviceList := []*v1.Service{
			tests.NewServiceFixture(),
		}
		cbCtx := &ConfigBuilderContext{
			IngressList:           cb.k8sContext.ListHTTPIngresses(),
			ServiceList:           serviceList,
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}
		_ = cb.RewriteRuleSets(cbCtx)

		It("should contain correct number of rewrite rule sets", func() {
			Expect(len(*cb.appGw.RewriteRuleSets)).To(Equal(0))
		})
	})

	Context("ensure correct removal of AGICGeneratedRewriteRuleSets", func() {
		inputRewriteRuleSet := []n.ApplicationGatewayRewriteRuleSet{
			{Name: to.StringPtr("crd-mynamespace-myrrs1")},
			{Name: to.StringPtr("myrrs2")},
			{Name: to.StringPtr("crd-mynamespace-myrrs3")},
			{Name: to.StringPtr("myrrs4")},
			{Name: to.StringPtr("crd-mynamespace-myrrs5")},
		}

		actualRewriteRuleSet := removeAGICGeneratedRewriteRuleSets(&inputRewriteRuleSet)

		expectedRewriteRuleSet := []n.ApplicationGatewayRewriteRuleSet{
			{Name: to.StringPtr("myrrs2")},
			{Name: to.StringPtr("myrrs4")},
		}

		It("should contain correct number of ApplicationGatewayRewriteRuleSets", func() {
			Expect(len(actualRewriteRuleSet)).To(Equal(len(expectedRewriteRuleSet)))
		})
		It("should contain correct set of ApplicationGatewayRewriteRuleSets", func() {
			Expect(actualRewriteRuleSet).To(Equal(expectedRewriteRuleSet))
		})
	})

	// The below test uses c.makeRewrite as well as makeConditions, makeActionSet, makeHeaderConfig, makeURLConfig
	Context("ensure correct conversion of *v1beta1.AzureApplicationGatewayRewrite to n.ApplicationGatewayRewriteRuleSet", func() {

		inputSpec := v1beta1.AzureApplicationGatewayRewrite{
			Spec: v1beta1.AzureApplicationGatewayRewriteSpec{
				RewriteRules: []v1beta1.RewriteRule{
					{
						Name:         "test-rewrite-rule-1",
						RuleSequence: 100,
						Actions: v1beta1.Actions{
							RequestHeaderConfigurations: []v1beta1.HeaderConfiguration{
								{
									ActionType:  "set",
									HeaderName:  "h1",
									HeaderValue: "v1",
								},
								{
									ActionType: "delete",
									HeaderName: "h2",
								},
							},
							ResponseHeaderConfigurations: []v1beta1.HeaderConfiguration{
								{
									ActionType:  "set",
									HeaderName:  "h3",
									HeaderValue: "v3",
								},
								{
									ActionType: "delete",
									HeaderName: "h4",
								},
							},
							UrlConfiguration: v1beta1.UrlConfiguration{
								ModifiedPath:        "abc",
								ModifiedQueryString: "def",
								Reroute:             false,
							},
						},
						Conditions: []v1beta1.Condition{
							{
								IgnoreCase: false,
								Negate:     false,
								Variable:   "aaa",
								Pattern:    "bbb",
							},
							{
								IgnoreCase: true,
								Negate:     true,
								Variable:   "ccc",
								Pattern:    "ddd",
							},
						},
					},
					{
						Name:         "test-rewrite-rule-2",
						RuleSequence: 101,
						Actions: v1beta1.Actions{
							RequestHeaderConfigurations: []v1beta1.HeaderConfiguration{
								{
									ActionType:  "set",
									HeaderName:  "h1",
									HeaderValue: "v1",
								},
								{
									ActionType: "delete",
									HeaderName: "h2",
								},
							},
							ResponseHeaderConfigurations: []v1beta1.HeaderConfiguration{
								{
									ActionType:  "set",
									HeaderName:  "h3",
									HeaderValue: "v3",
								},
								{
									ActionType: "delete",
									HeaderName: "h4",
								},
							},
							UrlConfiguration: v1beta1.UrlConfiguration{
								ModifiedPath:        "abc",
								ModifiedQueryString: "def",
								Reroute:             false,
							},
						},
						Conditions: []v1beta1.Condition{
							{
								IgnoreCase: false,
								Negate:     false,
								Variable:   "aaa",
								Pattern:    "bbb",
							},
							{
								IgnoreCase: true,
								Negate:     true,
								Variable:   "ccc",
								Pattern:    "ddd",
							},
						},
					},
				},
			},
		}

		ns := "mynamespace"
		actualRewriteRuleSetCRDName := "test-rewrite-rule-set"
		expectedRewriteRuleSetCRDName := fmt.Sprintf("crd-%s-%s", ns, actualRewriteRuleSetCRDName)

		cb := newConfigBuilderFixture(nil)
		actualOutput := cb.makeRewrite(ns, actualRewriteRuleSetCRDName, &inputSpec)

		expectedOuput := n.ApplicationGatewayRewriteRuleSet{
			Name: to.StringPtr(expectedRewriteRuleSetCRDName),
			ID:   to.StringPtr(cb.appGwIdentifier.rewriteRuleSetID(expectedRewriteRuleSetCRDName)),

			ApplicationGatewayRewriteRuleSetPropertiesFormat: &n.ApplicationGatewayRewriteRuleSetPropertiesFormat{
				RewriteRules: &[]n.ApplicationGatewayRewriteRule{
					{
						Name:         to.StringPtr("test-rewrite-rule-1"),
						RuleSequence: to.Int32Ptr(100),
						Conditions: &[]n.ApplicationGatewayRewriteRuleCondition{
							{
								Variable:   to.StringPtr("aaa"),
								Pattern:    to.StringPtr("bbb"),
								IgnoreCase: to.BoolPtr(false),
								Negate:     to.BoolPtr(false),
							},
							{
								Variable:   to.StringPtr("ccc"),
								Pattern:    to.StringPtr("ddd"),
								IgnoreCase: to.BoolPtr(true),
								Negate:     to.BoolPtr(true),
							},
						},
						ActionSet: &n.ApplicationGatewayRewriteRuleActionSet{
							RequestHeaderConfigurations: &[]n.ApplicationGatewayHeaderConfiguration{
								{
									HeaderName:  to.StringPtr("h1"),
									HeaderValue: to.StringPtr("v1"),
								},
								{
									HeaderName:  to.StringPtr("h2"),
									HeaderValue: to.StringPtr(""),
								},
							},
							ResponseHeaderConfigurations: &[]n.ApplicationGatewayHeaderConfiguration{
								{
									HeaderName:  to.StringPtr("h3"),
									HeaderValue: to.StringPtr("v3"),
								},
								{
									HeaderName:  to.StringPtr("h4"),
									HeaderValue: to.StringPtr(""),
								},
							},
							URLConfiguration: &n.ApplicationGatewayURLConfiguration{
								ModifiedPath:        to.StringPtr("abc"),
								ModifiedQueryString: to.StringPtr("def"),
								Reroute:             to.BoolPtr(false),
							},
						},
					},
					{
						Name:         to.StringPtr("test-rewrite-rule-2"),
						RuleSequence: to.Int32Ptr(101),
						Conditions: &[]n.ApplicationGatewayRewriteRuleCondition{
							{
								Variable:   to.StringPtr("aaa"),
								Pattern:    to.StringPtr("bbb"),
								IgnoreCase: to.BoolPtr(false),
								Negate:     to.BoolPtr(false),
							},
							{
								Variable:   to.StringPtr("ccc"),
								Pattern:    to.StringPtr("ddd"),
								IgnoreCase: to.BoolPtr(true),
								Negate:     to.BoolPtr(true),
							},
						},
						ActionSet: &n.ApplicationGatewayRewriteRuleActionSet{
							RequestHeaderConfigurations: &[]n.ApplicationGatewayHeaderConfiguration{
								{
									HeaderName:  to.StringPtr("h1"),
									HeaderValue: to.StringPtr("v1"),
								},
								{
									HeaderName:  to.StringPtr("h2"),
									HeaderValue: to.StringPtr(""),
								},
							},
							ResponseHeaderConfigurations: &[]n.ApplicationGatewayHeaderConfiguration{
								{
									HeaderName:  to.StringPtr("h3"),
									HeaderValue: to.StringPtr("v3"),
								},
								{
									HeaderName:  to.StringPtr("h4"),
									HeaderValue: to.StringPtr(""),
								},
							},
							URLConfiguration: &n.ApplicationGatewayURLConfiguration{
								ModifiedPath:        to.StringPtr("abc"),
								ModifiedQueryString: to.StringPtr("def"),
								Reroute:             to.BoolPtr(false),
							},
						},
					},
				},
				ProvisioningState: "",
			},
		}

		It("should contain correct numbder ApplicationGatewayRewriteRuleSet", func() {
			Expect(len(*actualOutput.RewriteRules)).To(Equal(len(*expectedOuput.RewriteRules)))
		})

		It("should contain correct ApplicationGatewayRewriteRuleSet", func() {
			Expect(actualOutput).To(Equal(expectedOuput))
		})
	})
})
