// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test routing rules generations", func() {
	Context("test path-based rule with 2 ingress both with paths", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingressPathBased1 := tests.NewIngressFixture()
		ingressPathBased1.Annotations[annotations.SslRedirectKey] = "false"
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased1)

		ingressPathBased2 := tests.NewIngressFixture()
		ingressPathBased2.Name = "ingress1"
		ingressPathBased2.Annotations[annotations.SslRedirectKey] = "false"
		testEndpoint := tests.NewEndpointsFixture()
		testEndpoint.Name = "test"
		testService := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		testService.Name = "test"
		testBackend := tests.NewIngressBackendFixture("test", 80)
		testRule := tests.NewIngressRuleFixture(tests.Host, tests.URLPath3, *testBackend)
		ingressPathBased2.Spec.Rules = []v1beta1.IngressRule{
			testRule,
		}
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(testEndpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(testService)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased2)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingressPathBased1, ingressPathBased2},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)
		_ = configBuilder.RequestRoutingRules(cbCtx)

		rule := &ingressPathBased1.Spec.Rules[0]

		_ = configBuilder.Listeners(cbCtx)
		// !! Action !! -- will mutate pathMap struct
		pathMaps := configBuilder.getPathMaps(cbCtx)
		sharedListenerID := generateListenerID(ingressPathBased1, rule, n.HTTPS, nil, false)
		generatedPathMap := pathMaps[sharedListenerID]
		It("has default backend pool", func() {
			Expect(generatedPathMap.DefaultBackendAddressPool).To(Not(BeNil()))
		})
		It("has default backend http settings", func() {
			Expect(generatedPathMap.DefaultBackendHTTPSettings).To(Not(BeNil()))
		})
		It("should has 3 path rules", func() {
			Expect(len(*generatedPathMap.PathRules)).To(Equal(3))
		})
		It("should be able to merge all the path rules into the same path map", func() {
			for _, ingress := range cbCtx.IngressList {
				for _, rule := range ingress.Spec.Rules {
					for _, path := range rule.HTTP.Paths {
						backendID := generateBackendID(ingress, &rule, &path, &path.Backend)
						backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), Port(tests.ContainerPort)))
						httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), Port(tests.ContainerPort), backendID.Ingress.Name))
						pathRuleName := generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, "0")
						expectedPathRule := n.ApplicationGatewayPathRule{
							Name: to.StringPtr(pathRuleName),
							Etag: to.StringPtr("*"),
							ID:   to.StringPtr(configBuilder.appGwIdentifier.pathRuleID(*generatedPathMap.Name, pathRuleName)),
							ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
								Paths: &[]string{
									path.Path,
								},
								BackendAddressPool:  &n.SubResource{ID: to.StringPtr(backendPoolID)},
								BackendHTTPSettings: &n.SubResource{ID: to.StringPtr(httpSettingID)},
							},
						}
						Expect(*generatedPathMap.PathRules).To(ContainElement(expectedPathRule))
					}
				}
			}
		})
	})

	Context("test path-based rule with 2 ingress both with paths", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)

		// 2 path based rules
		ingressPathBased := tests.NewIngressFixture()
		ingressPathBased.Annotations[annotations.SslRedirectKey] = "false"

		// 1 basic rule
		ingressBasic := tests.NewIngressFixture()
		ingressBasic.Name = "ingressBasic"
		ingressBasic.Annotations[annotations.SslRedirectKey] = "false"
		backendBasic := tests.NewIngressBackendFixture(tests.ServiceName, 80)
		ruleBasic := tests.NewIngressRuleFixture(tests.Host, "", *backendBasic)
		pathBasic := &ruleBasic.HTTP.Paths[0]
		ingressBasic.Spec.Rules = []v1beta1.IngressRule{
			ruleBasic,
		}

		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressBasic)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingressPathBased, ingressBasic},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)
		_ = configBuilder.RequestRoutingRules(cbCtx)

		rule := &ingressPathBased.Spec.Rules[0]

		_ = configBuilder.Listeners(cbCtx)
		// !! Action !! -- will mutate pathMap struct
		pathMaps := configBuilder.getPathMaps(cbCtx)
		sharedListenerID := generateListenerID(ingressPathBased, rule, n.HTTPS, nil, false)
		generatedPathMap := pathMaps[sharedListenerID]
		backendIDBasic := generateBackendID(ingressBasic, &ruleBasic, pathBasic, backendBasic)
		It("has default backend pool coming from basic ingress", func() {
			backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendIDBasic.serviceFullName(), backendIDBasic.Backend.ServicePort.String(), Port(tests.ContainerPort)))
			Expect(*generatedPathMap.DefaultBackendAddressPool.ID).To(Equal(backendPoolID))
		})
		It("has default backend http settings coming from basic ingress", func() {
			httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendIDBasic.serviceFullName(), backendIDBasic.Backend.ServicePort.String(), Port(tests.ContainerPort), ingressBasic.Name))
			Expect(*generatedPathMap.DefaultBackendHTTPSettings.ID).To(Equal(httpSettingID))
		})
		It("should has 2 path rules", func() {
			Expect(len(*generatedPathMap.PathRules)).To(Equal(2))
		})
		It("should have two path rules coming from path based ingress", func() {
			for _, rule := range ingressPathBased.Spec.Rules {
				for _, path := range rule.HTTP.Paths {
					backendID := generateBackendID(ingressPathBased, &rule, &path, &path.Backend)
					backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), Port(tests.ContainerPort)))
					httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), Port(tests.ContainerPort), backendID.Ingress.Name))
					pathRuleName := generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, "0")
					expectedPathRule := n.ApplicationGatewayPathRule{
						Name: to.StringPtr(pathRuleName),
						ID:   to.StringPtr(configBuilder.appGwIdentifier.pathRuleID(*generatedPathMap.Name, pathRuleName)),
						Etag: to.StringPtr("*"),
						ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
							Paths: &[]string{
								path.Path,
							},
							BackendAddressPool:  &n.SubResource{ID: to.StringPtr(backendPoolID)},
							BackendHTTPSettings: &n.SubResource{ID: to.StringPtr(httpSettingID)},
						},
					}
					Expect(*generatedPathMap.PathRules).To(ContainElement(expectedPathRule))
				}
			}
		})
	})

	Context("test ssl redirect is configured correctly when a path based rule is created", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingress := tests.NewIngressFixture()

		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress)

		cbCtx := &ConfigBuilderContext{
			IngressList: []*v1beta1.Ingress{ingress},
			ServiceList: []*v1.Service{service},
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)
		_ = configBuilder.RequestRoutingRules(cbCtx)

		It("should have ingress with TLS and redirect", func() {
			Expect(len(ingress.Spec.TLS) != 0).To(BeTrue())
			Expect(ingress.Annotations[annotations.SslRedirectKey]).To(Equal("true"))
		})

		rule := &ingress.Spec.Rules[0]

		_ = configBuilder.Listeners(cbCtx)
		// !! Action !! -- will mutate pathMap struct
		pathMap := configBuilder.getPathMaps(cbCtx)
		listenerID := generateListenerID(ingress, rule, n.HTTP, nil, false)
		It("has no default backend pool", func() {
			Expect(pathMap[listenerID].DefaultBackendAddressPool).To(BeNil())
		})
		It("has no default backend http settings", func() {
			Expect(pathMap[listenerID].DefaultBackendHTTPSettings).To(BeNil())
		})

		expectedListenerID, _ := newTestListenerID(Port(443), []string{rule.Host}, false)
		expectedRedirectID := configBuilder.appGwIdentifier.redirectConfigurationID(
			generateSSLRedirectConfigurationName(expectedListenerID))
		actualID := *(pathMap[listenerID].DefaultRedirectConfiguration.ID)
		It("generated expected ID", func() {
			Expect(actualID).To(Equal(expectedRedirectID))
		})
		It("should still have 2 path rules", func() {
			Expect(2).To(Equal(len(*pathMap[listenerID].PathRules)))
		})
	})

	Context("test waf policy is configured in rule path", func() {
		configBuilder := newConfigBuilderFixture(nil)
		secret := tests.NewSecretTestFixture()
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingress := tests.NewIngressFixture()
		ingress.Annotations[annotations.FirewallPolicy] = "/sub/waf"

		It("should have ingress with TLS and redirect", func() {
			Expect(len(ingress.Spec.TLS) != 0).To(BeTrue())
			Expect(len(ingress.Spec.TLS[0].SecretName) != 0).To(BeTrue())
			Expect(ingress.Annotations[annotations.SslRedirectKey]).To(Equal("true"))
		})

		_ = configBuilder.k8sContext.Caches.Secret.Add(secret)
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)

		// !! Action !! -- will mutate pathMap struct
		pathMap := configBuilder.getPathMaps(cbCtx)

		rule := &ingress.Spec.Rules[0]
		listenerID := generateListenerID(ingress, rule, n.HTTP, nil, false)
		It("has waf policy in pathRule", func() {
			prs := pathMap[listenerID].ApplicationGatewayURLPathMapPropertiesFormat
			for _, r := range *prs.PathRules {
				Expect(r.FirewallPolicy.ID).To(Equal(to.StringPtr("/sub/waf")))
			}
		})
	})

	Context("test waf policy is not configured in rule path", func() {
		configBuilder := newConfigBuilderFixture(nil)
		secret := tests.NewSecretTestFixture()
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingress := tests.NewIngressTestFixtureBasic(tests.Namespace, "random", false)
		ingress.Annotations[annotations.FirewallPolicy] = "/sub/waf"

		_ = configBuilder.k8sContext.Caches.Secret.Add(secret)
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)

		// !! Action !! -- will mutate pathMap struct
		pathMap := configBuilder.getPathMaps(cbCtx)

		rule := &ingress.Spec.Rules[0]
		listenerID := generateListenerID(ingress, rule, n.HTTP, nil, false)
		It("has no waf policy in pathRule", func() {
			prs := pathMap[listenerID].ApplicationGatewayURLPathMapPropertiesFormat
			Expect(prs.PathRules).To(BeNil())
		})
	})

	Context("test ssl redirect is configured correctly when a basic rule is created", func() {
		configBuilder := newConfigBuilderFixture(nil)
		secret := tests.NewSecretTestFixture()
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingress := tests.NewIngressTestFixtureBasic(tests.Namespace, "random", true)

		It("should have ingress with TLS and redirect", func() {
			Expect(len(ingress.Spec.TLS) != 0).To(BeTrue())
			Expect(len(ingress.Spec.TLS[0].SecretName) != 0).To(BeTrue())
			Expect(ingress.Annotations[annotations.SslRedirectKey]).To(Equal("true"))
		})

		_ = configBuilder.k8sContext.Caches.Secret.Add(secret)
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)

		// !! Action !! -- will mutate pathMap struct
		pathMap := configBuilder.getPathMaps(cbCtx)

		rule := &ingress.Spec.Rules[0]
		listenerID := generateListenerID(ingress, rule, n.HTTP, nil, false)
		It("has no default backend pool", func() {
			Expect(pathMap[listenerID].DefaultBackendAddressPool).To(BeNil())
		})
		It("has no default backend http settings", func() {
			Expect(pathMap[listenerID].DefaultBackendHTTPSettings).To(BeNil())
		})
		It("has no pathrules", func() {
			Expect(pathMap[listenerID].PathRules).To(BeNil())
		})

		expectedListenerID, _ := newTestListenerID(Port(443), []string{rule.Host}, false)
		expectedRedirectID := configBuilder.appGwIdentifier.redirectConfigurationID(
			generateSSLRedirectConfigurationName(expectedListenerID))
		actualID := *(pathMap[listenerID].DefaultRedirectConfiguration.ID)
		It("generated expected ID", func() {
			Expect(expectedRedirectID).To(Equal(actualID))
		})
	})

	Context("test RequestRoutingRules without HTTPS but with SSL Redirect", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingress := fixtures.GetIngress()

		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)
		_ = configBuilder.RequestRoutingRules(cbCtx)

		expectedListenerID80, expectedListenerID80Name := newTestListenerID(Port(80), []string{"foo.baz"}, false)
		expectedListenerID443, expectedListenerID443Name := newTestListenerID(Port(443), []string{"foo.baz"}, false)
		It("should have correct RequestRoutingRules", func() {
			Expect(len(*configBuilder.appGw.RequestRoutingRules)).To(Equal(2))

			Expect(*configBuilder.appGw.RequestRoutingRules).To(ContainElement(n.ApplicationGatewayRequestRoutingRule{
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType:            "Basic",
					BackendAddressPool:  nil,
					BackendHTTPSettings: nil,
					HTTPListener: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/" + expectedListenerID80Name),
					},
					URLPathMap:     nil,
					RewriteRuleSet: nil,
					RedirectConfiguration: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
							"/redirectConfigurations/sslr-" + expectedListenerID443Name)},
					ProvisioningState: "",
				},
				Name: to.StringPtr("rr-" + utils.GetHashCode(expectedListenerID80)),
				Etag: to.StringPtr("*"),
				Type: nil,
				ID:   to.StringPtr(configBuilder.appGwIdentifier.requestRoutingRuleID("rr-" + utils.GetHashCode(expectedListenerID80))),
			}))

			Expect(*configBuilder.appGw.RequestRoutingRules).To(ContainElement(n.ApplicationGatewayRequestRoutingRule{
				ApplicationGatewayRequestRoutingRulePropertiesFormat: &n.ApplicationGatewayRequestRoutingRulePropertiesFormat{
					RuleType: "Basic",
					BackendAddressPool: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
							"/backendAddressPools/pool---namespace-----service-name---80-bp-9876"),
					},
					BackendHTTPSettings: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
							"/backendHttpSettingsCollection/bp---namespace-----service-name---80-9876---name--"),
					},
					HTTPListener: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/" + expectedListenerID443Name),
					},
					URLPathMap:            nil,
					RewriteRuleSet:        nil,
					RedirectConfiguration: nil,
					ProvisioningState:     "",
				},
				Name: to.StringPtr("rr-" + utils.GetHashCode(expectedListenerID443)),
				Etag: to.StringPtr("*"),
				Type: nil,
				ID:   to.StringPtr(configBuilder.appGwIdentifier.requestRoutingRuleID("rr-" + utils.GetHashCode(expectedListenerID443))),
			}))
		})

		It("should have correct URLPathMaps", func() {
			Expect(len(*configBuilder.appGw.URLPathMaps)).To(Equal(0))
		})
	})
})
