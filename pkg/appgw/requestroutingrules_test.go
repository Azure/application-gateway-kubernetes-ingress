// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test routing rules generations", func() {
	defer GinkgoRecover()

	checkPathRules := func(urlPathMap *n.ApplicationGatewayURLPathMap, pathRuleCount int) {
		if pathRuleCount == 0 {
			Expect(urlPathMap.PathRules).To(BeNil())
		}

		Expect(len(*urlPathMap.PathRules)).To(Equal(pathRuleCount))

		// check name uniqueness
		nameMap := map[string]interface{}{}
		for _, pathRule := range *urlPathMap.PathRules {
			_, exists := nameMap[*pathRule.Name]
			Expect(exists).To(BeFalse())
			nameMap[*pathRule.Name] = nil
		}
	}

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
		ingressPathBased2.Spec.Rules = []networking.IngressRule{
			testRule,
		}
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(testEndpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(testService)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased2)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingressPathBased1, ingressPathBased2},
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
		sharedListenerID := generateListenerID(ingressPathBased1, rule, n.ApplicationGatewayProtocolHTTPS, nil, false)
		generatedPathMap := pathMaps[sharedListenerID]
		It("has default backend pool", func() {
			Expect(generatedPathMap.DefaultBackendAddressPool).To(Not(BeNil()))
		})
		It("has default backend http settings", func() {
			Expect(generatedPathMap.DefaultBackendHTTPSettings).To(Not(BeNil()))
		})
		It("should have uniquely names path rules and 3 path rules", func() {
			checkPathRules(generatedPathMap, 3)
		})
		It("should be able to merge all the path rules into the same path map", func() {
			for _, ingress := range cbCtx.IngressList {
				for ruleIdx, rule := range ingress.Spec.Rules {
					for pathIdx, path := range rule.HTTP.Paths {
						backendID := generateBackendID(ingress, &rule, &path, &path.Backend)
						backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort)))
						httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort), backendID.Ingress.Name))
						pathRuleName := generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, ruleIdx, pathIdx)
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
		ingressBasic.Spec.Rules = []networking.IngressRule{
			ruleBasic,
		}

		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressBasic)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingressPathBased, ingressBasic},
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
		sharedListenerID := generateListenerID(ingressPathBased, rule, n.ApplicationGatewayProtocolHTTPS, nil, false)
		generatedPathMap := pathMaps[sharedListenerID]
		backendIDBasic := generateBackendID(ingressBasic, &ruleBasic, pathBasic, backendBasic)
		It("has default backend pool coming from basic ingress", func() {
			backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendIDBasic.serviceFullName(), serviceBackendPortToStr(backendIDBasic.Backend.Service.Port), Port(tests.ContainerPort)))
			Expect(*generatedPathMap.DefaultBackendAddressPool.ID).To(Equal(backendPoolID))
		})
		It("has default backend http settings coming from basic ingress", func() {
			httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendIDBasic.serviceFullName(), serviceBackendPortToStr(backendIDBasic.Backend.Service.Port), Port(tests.ContainerPort), ingressBasic.Name))
			Expect(*generatedPathMap.DefaultBackendHTTPSettings.ID).To(Equal(httpSettingID))
		})
		It("should have uniquely names path rules and has 2 path rules", func() {
			checkPathRules(generatedPathMap, 2)
		})
		It("should have two path rules coming from path based ingress", func() {
			for ruleIdx, rule := range ingressPathBased.Spec.Rules {
				for pathIdx, path := range rule.HTTP.Paths {
					backendID := generateBackendID(ingressPathBased, &rule, &path, &path.Backend)
					backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort)))
					httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort), backendID.Ingress.Name))
					pathRuleName := generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, ruleIdx, pathIdx)
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
			IngressList: []*networking.Ingress{ingress},
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
		listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTP, nil, false)
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
		It("should have uniquely names path rules and still has 2 path rules", func() {
			checkPathRules(pathMap[listenerID], 2)
		})
	})

	Context("test override frontend port", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingress := tests.NewIngressFixture()

		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress)

		cbCtx := &ConfigBuilderContext{
			IngressList: []*networking.Ingress{ingress},
			ServiceList: []*v1.Service{service},
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)
		_ = configBuilder.RequestRoutingRules(cbCtx)

		rule := &ingress.Spec.Rules[0]

		It("frontend port is default to 80", func() {
			listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTP, nil, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(80)))
		})

		It("frontend port is default to 80 when no annotation", func() {
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTP, &overrideFrontendPort, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(80)))
		})

		It("frontend port is default to 443 when https", func() {
			listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTPS, nil, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(443)))
		})

		It("frontend port is default to 443 when https with no annotation", func() {
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTPS, &overrideFrontendPort, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(443)))
		})

		It("frontend port is overridden in annotation", func() {
			ingress.Annotations[annotations.OverrideFrontendPortKey] = "777"
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTP, &overrideFrontendPort, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(777)))
		})

		It("frontend port is out of range", func() {
			ingress.Annotations[annotations.OverrideFrontendPortKey] = "65000"
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTP, &overrideFrontendPort, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(80)))
		})

		It("frontend port is out of range", func() {
			ingress.Annotations[annotations.OverrideFrontendPortKey] = "0"
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTP, &overrideFrontendPort, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(80)))
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
			IngressList:           []*networking.Ingress{ingress},
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
		listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTP, nil, false)
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
			IngressList:           []*networking.Ingress{ingress},
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
		listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTP, nil, false)
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
			IngressList:           []*networking.Ingress{ingress},
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
		listenerID := generateListenerID(ingress, rule, n.ApplicationGatewayProtocolHTTP, nil, false)
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
			IngressList:           []*networking.Ingress{ingress},
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
					Priority:          to.Int32Ptr(19000),
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
					Priority:              to.Int32Ptr(19005),
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

	Context("test path-based rule with 1 ingress and multiple service having with duplicate url paths", func() {
		configBuilder := newConfigBuilderFixture(nil)
		testServiceName := "testService"
		endpoint1 := tests.NewEndpointsFixture()
		endpoint2 := tests.NewEndpointsFixture()
		endpoint2.Name = testServiceName
		service1 := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		service2 := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		service2.Name = testServiceName

		// 2 path based rules with path - /api1, /api2
		ingressPathBased := tests.NewIngressFixture()
		ingressPathBased.Annotations[annotations.SslRedirectKey] = "true"
		backendBasic := tests.NewIngressBackendFixture(service2.Name, 80)

		// Adding duplicate path /api for a different service in same ingress
		duplicatePathRule := tests.NewIngressRuleFixture(tests.Host, "/api1", *backendBasic)
		ingressPathBased.Spec.Rules = append([]networking.IngressRule{duplicatePathRule}, ingressPathBased.Spec.Rules...)

		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint1)
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint2)
		_ = configBuilder.k8sContext.Caches.Service.Add(service1)
		_ = configBuilder.k8sContext.Caches.Service.Add(service2)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingressPathBased},
			ServiceList:           []*v1.Service{service1, service2},
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
		sharedListenerID := generateListenerID(ingressPathBased, rule, n.ApplicationGatewayProtocolHTTPS, nil, false)
		generatedPathMap := pathMaps[sharedListenerID]
		It("has default backend pool coming from path-based ingress", func() {
			Expect(*generatedPathMap.DefaultBackendAddressPool.ID).To(Equal("xx"))
		})
		It("has default backend http settings coming from basic ingress", func() {
			Expect(*generatedPathMap.DefaultBackendHTTPSettings.ID).To(Equal("yy"))
		})
		It("should has 2 path rules", func() {
			Expect(len(*generatedPathMap.PathRules)).To(Equal(2))
		})
		It("should have two path rules coming from path based ingress", func() {
			for ruleIdx, rule := range ingressPathBased.Spec.Rules {
				for pathIdx, path := range rule.HTTP.Paths {
					backendID := generateBackendID(ingressPathBased, &rule, &path, &path.Backend)
					backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort)))
					httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort), backendID.Ingress.Name))
					pathRuleName := generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, ruleIdx, pathIdx)
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

					if ruleIdx == 1 {
						// this is the second rule for /api1 which should not be added to the path rules
						Expect(*generatedPathMap.PathRules).ToNot(ContainElement(expectedPathRule), fmt.Sprintf("%+v", ingressPathBased))
					} else {
						Expect(*generatedPathMap.PathRules).To(ContainElement(expectedPathRule), fmt.Sprintf("%+v", ingressPathBased))
					}
				}
			}
		})
	})

	Context("test path-based rule with 1 ingress and 1 service having 1 rule with duplicate paths", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint1 := tests.NewEndpointsFixture()
		service1 := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)

		// 2 path based rules with path - /api1, /api2
		ingressPathBased := tests.NewIngressFixture()
		ingressPathBased.Annotations[annotations.SslRedirectKey] = "true"
		backendBasic := tests.NewIngressBackendFixture(tests.ServiceName, 80)
		// Adding duplicate path /api3 for a same service in same ingress
		duplicatePathRule := tests.NewIngressRuleWithPathsFixture(tests.Host, []string{"/api3", "/api3"}, *backendBasic)
		ingressPathBased.Spec.Rules = append([]networking.IngressRule{duplicatePathRule}, ingressPathBased.Spec.Rules...)

		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint1)
		_ = configBuilder.k8sContext.Caches.Service.Add(service1)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingressPathBased},
			ServiceList:           []*v1.Service{service1},
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
		sharedListenerID := generateListenerID(ingressPathBased, rule, n.ApplicationGatewayProtocolHTTPS, nil, false)
		generatedPathMap := pathMaps[sharedListenerID]
		It("has default backend pool coming from path-based ingress", func() {
			Expect(*generatedPathMap.DefaultBackendAddressPool.ID).To(Equal("xx"))
		})
		It("has default backend http settings coming from basic ingress", func() {
			Expect(*generatedPathMap.DefaultBackendHTTPSettings.ID).To(Equal("yy"))
		})
		It("should has 3 path rules", func() {
			// even though there are 4 paths, namely /api1, /api2, /api3, /api3,
			// the generated pathmap count will be 3 after removing duplicates
			Expect(len(*generatedPathMap.PathRules)).To(Equal(3))
		})
		It("should have three path rules coming from path based ingress", func() {
			for ruleIdx, rule := range ingressPathBased.Spec.Rules {
				pathMapPerRule := make(map[string]bool)
				for pathIdx, path := range rule.HTTP.Paths {
					backendID := generateBackendID(ingressPathBased, &rule, &path, &path.Backend)
					backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort)))
					httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort), backendID.Ingress.Name))
					pathRuleName := generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, ruleIdx, pathIdx)
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

					if _, exists := pathMapPerRule[path.Path]; exists {
						// duplicate paths in a rule are not accepted
						Expect(*generatedPathMap.PathRules).ToNot(ContainElement(expectedPathRule))
					} else {
						pathMapPerRule[path.Path] = true
						Expect(*generatedPathMap.PathRules).To(ContainElement(expectedPathRule))
					}
				}
			}
		})
	})

	Context("test path-based rule with 2 ingress both with having same paths", func() {
		// Since 2 ingress are created with same hostname all path rules for ingress merge because of same listenerId
		// In case of duplicate rules in 2 ingress, only rules from first ingress will be part of the path rules
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
		testRule := tests.NewIngressRuleFixture(tests.Host, tests.URLPath1, *testBackend)
		ingressPathBased2.Spec.Rules = []networking.IngressRule{
			testRule,
		}
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(testEndpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(testService)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased2)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingressPathBased1, ingressPathBased2},
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
		sharedListenerID := generateListenerID(ingressPathBased1, rule, n.ApplicationGatewayProtocolHTTPS, nil, false)
		generatedPathMap := pathMaps[sharedListenerID]
		It("has default backend pool", func() {
			Expect(generatedPathMap.DefaultBackendAddressPool).To(Not(BeNil()))
		})
		It("has default backend http settings", func() {
			Expect(generatedPathMap.DefaultBackendHTTPSettings).To(Not(BeNil()))
		})
		It("should have uniquely names path rules and 2 path rules", func() {
			checkPathRules(generatedPathMap, 2)
		})
		It("should be able to merge all the path rules into the same path map", func() {
			ingress := cbCtx.IngressList[0]
			for ruleIdx, rule := range ingress.Spec.Rules {
				for pathIdx, path := range rule.HTTP.Paths {
					backendID := generateBackendID(ingress, &rule, &path, &path.Backend)
					backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort)))
					httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort), backendID.Ingress.Name))
					pathRuleName := generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, ruleIdx, pathIdx)
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

			ingress = cbCtx.IngressList[1]
			for ruleIdx, rule := range ingress.Spec.Rules {
				for pathIdx, path := range rule.HTTP.Paths {
					backendID := generateBackendID(ingress, &rule, &path, &path.Backend)
					backendPoolID := configBuilder.appGwIdentifier.AddressPoolID(generateAddressPoolName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort)))
					httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), serviceBackendPortToStr(backendID.Backend.Service.Port), Port(tests.ContainerPort), backendID.Ingress.Name))
					pathRuleName := generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, ruleIdx, pathIdx)
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
					Expect(*generatedPathMap.PathRules).ToNot(ContainElement(expectedPathRule))
				}
			}
		})
	})

	Context("test ingress rewrite rule set with 2 ingresses one with rule set and another without", func() {
		configBuilder := newConfigBuilderFixture(nil)
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingressPathBased1 := tests.NewIngressFixture()
		rewriteRuleSetName := "custom-response-header"
		ingressPathBased1.Annotations[annotations.RewriteRuleSetKey] = rewriteRuleSetName

		ingressPathBased2 := tests.NewIngressFixture()
		testBackend := tests.NewIngressBackendFixture("test", 80)
		testRule := tests.NewIngressRuleFixture(tests.Host, tests.URLPath3, *testBackend)
		ingressPathBased2.Spec.Rules = []networking.IngressRule{
			testRule,
		}

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingressPathBased1, ingressPathBased2},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = configBuilder.Listeners(cbCtx)

		pathMaps := configBuilder.getPathMaps(cbCtx)

		sharedRule := &ingressPathBased1.Spec.Rules[0]

		sharedListenerID := generateListenerID(ingressPathBased1, sharedRule, n.ApplicationGatewayProtocolHTTPS, nil, false)

		It("has pathrules", func() {
			Expect(*pathMaps[sharedListenerID].PathRules).To(Not(BeNil()))
		})
		It("has exactly three path rule", func() {
			Expect(len(*pathMaps[sharedListenerID].PathRules)).To(Equal(3))
		})
		expectedRewriteRuleSet := resourceRef(configBuilder.appGwIdentifier.rewriteRuleSetID(rewriteRuleSetName))

		// the paths defined in both ingresses have rewrite rules since the annotation in the first ingress takes
		// precendence
		It("has rewrite rule set in first path rule", func() {
			Expect((*pathMaps[sharedListenerID].PathRules)[0].RewriteRuleSet).To(Equal(expectedRewriteRuleSet))
		})
		It("has rewrite rule set in second path rule", func() {
			Expect((*pathMaps[sharedListenerID].PathRules)[1].RewriteRuleSet).To(Equal(expectedRewriteRuleSet))
		})

		// the path that is only declared in ingress2 doesn't have rewrite rules
		It("has no rewrite rule set", func() {
			Expect((*pathMaps[sharedListenerID].PathRules)[2].RewriteRuleSet).To(BeNil())
		})
	})

	Context("test ingress rewrite rule set with two ingresses with different rule sets", func() {
		configBuilder := newConfigBuilderFixture(nil)
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingressPathBased1 := tests.NewIngressFixture()
		rewriteRuleSetName1 := "custom-response-header1"
		ingressPathBased1.Annotations[annotations.RewriteRuleSetKey] = rewriteRuleSetName1

		ingressPathBased2 := tests.NewIngressFixture()
		rewriteRuleSetName2 := "custom-response-header2"
		ingressPathBased2.Annotations[annotations.RewriteRuleSetKey] = rewriteRuleSetName2
		testBackend := tests.NewIngressBackendFixture("test", 80)
		testRule := tests.NewIngressRuleFixture(tests.Host, tests.URLPath3, *testBackend)

		ingressPathBased2.Spec.Rules = []networking.IngressRule{
			testRule,
		}

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingressPathBased1, ingressPathBased2},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_ = configBuilder.Listeners(cbCtx)

		pathMaps := configBuilder.getPathMaps(cbCtx)

		sharedRule := &ingressPathBased1.Spec.Rules[0]

		sharedListenerID := generateListenerID(ingressPathBased1, sharedRule, n.ApplicationGatewayProtocolHTTPS, nil, false)

		It("has pathrules", func() {
			Expect(*pathMaps[sharedListenerID].PathRules).To(Not(BeNil()))
		})
		It("has exactly three path rule", func() {
			Expect(len(*pathMaps[sharedListenerID].PathRules)).To(Equal(3))
		})
		expectedRewriteRuleSet1 := resourceRef(configBuilder.appGwIdentifier.rewriteRuleSetID(rewriteRuleSetName1))

		// the paths defined in both ingresses have rewrite rules declared in the first ingress since it takes
		// precendence
		It("has rewrite rule set in first path rule", func() {
			Expect((*pathMaps[sharedListenerID].PathRules)[0].RewriteRuleSet).To(Equal(expectedRewriteRuleSet1))
		})
		It("has rewrite rule set in second path rule", func() {
			Expect((*pathMaps[sharedListenerID].PathRules)[1].RewriteRuleSet).To(Equal(expectedRewriteRuleSet1))
		})

		expectedRewriteRuleSet2 := resourceRef(configBuilder.appGwIdentifier.rewriteRuleSetID(rewriteRuleSetName2))
		// the path that is only declared in ingress2 has the rewrite rule declared in ingress2
		It("has rewrite rule set from ingress 2", func() {
			Expect((*pathMaps[sharedListenerID].PathRules)[2].RewriteRuleSet).To(Equal(expectedRewriteRuleSet2))
		})
	})

	Context("test ingress rewrite rule set in basic ingress", func() {
		configBuilder := newConfigBuilderFixture(nil)
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingress := tests.NewIngressTestFixtureBasic(tests.Namespace, "random", false)
		rewriteRuleSetName := "custom-response-header"
		ingress.Annotations[annotations.RewriteRuleSetKey] = rewriteRuleSetName

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingress},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		requestRoutingRules, _ := configBuilder.getRules(cbCtx)

		expectedRewriteRuleSet := resourceRef(configBuilder.appGwIdentifier.rewriteRuleSetID(rewriteRuleSetName))

		It("has rewrite rule set", func() {
			Expect(requestRoutingRules[0].RewriteRuleSet).To(Equal(expectedRewriteRuleSet))
		})

	})

	Context("test pathType in ingress", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingress := tests.NewIngressTestWithVariousPathTypeFixture(tests.Namespace, "ingress-with-path-type")

		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{&ingress},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		_, urlPathMaps := configBuilder.getRules(cbCtx)
		urlPathMap := urlPathMaps[0]
		pathRules := *urlPathMaps[0].PathRules

		It("should have 8 paths", func() {
			Expect(len(pathRules)).To(Equal(8))
		})

		It("should add * to paths with pathType:prefix", func() {
			paths := *(pathRules[0].Paths)
			Expect(paths[0]).To(Equal("/prefix0*"))

			paths = *(pathRules[1].Paths)
			Expect(paths[0]).To(Equal("/prefix1*"))
		})

		It("should trim * from paths with pathType:exact", func() {
			paths := *(pathRules[2].Paths)
			Expect(paths[0]).To(Equal("/exact2"))

			paths = *(pathRules[3].Paths)
			Expect(paths[0]).To(Equal("/exact3"))
		})

		It("should not modify paths with pathType:implementationSpecific", func() {
			paths := *(pathRules[4].Paths)
			Expect(paths[0]).To(Equal("/ims4*"))

			paths = *(pathRules[5].Paths)
			Expect(paths[0]).To(Equal("/ims5"))
		})

		It("should not modify paths with pathType:nil", func() {
			paths := *(pathRules[6].Paths)
			Expect(paths[0]).To(Equal("/nil6*"))

			paths = *(pathRules[7].Paths)
			Expect(paths[0]).To(Equal("/nil7"))
		})

		It("should have default matching /*", func() {
			Expect(*urlPathMap.DefaultBackendAddressPool.ID).To(HaveSuffix("pool---namespace-----service-name---80-bp-9876"))
			Expect(*urlPathMap.DefaultBackendHTTPSettings.ID).To(HaveSuffix("bp---namespace-----service-name---80-9876-ingress-with-path-type"))
		})
	})

	Describe("test auto assigning missing routing rule prirority", func() {
		var configBuilder appGwConfigBuilder
		var ingress *networking.Ingress
		var cbCtx *ConfigBuilderContext
		// var prohibitedTargets []*ptv1.AzureIngressProhibitedTarget
		var ruleCount int
		var ingresses []*networking.Ingress

		BeforeEach(func() {
			ingresses = make([]*networking.Ingress, 0)
			configBuilder = newConfigBuilderFixture(nil)
			backend := tests.NewIngressBackendFixture(tests.ServiceName, int32(80))

			endpoint := tests.NewEndpointsFixture()
			service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)

			Expect(configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)).To(Succeed())
			Expect(configBuilder.k8sContext.Caches.Service.Add(service)).To(Succeed())

			// The upper bound for rule priority defined as 20000 in NRP.
			// With multisite listeners, we auto generate priority starting from
			// 19000 with an increment of 10. Thus, AGIC's bahivor in
			// scenarios where customers define over 50 rules for basic
			// listeners is undefined.
			ruleCount = 50
			for i := 0; i < ruleCount; i++ {
				ingress = &networking.Ingress{}
				ingress.Name = fmt.Sprint(i)
				ingress.Namespace = tests.Namespace
				ingress.Annotations = map[string]string{
					annotations.OverrideFrontendPortKey: fmt.Sprint(i),
					annotations.IngressClassKey:         tests.IngressClassController,
				}
				rule := tests.NewIngressRuleFixture("", "/", *backend)
				ingress.Spec.Rules = []networking.IngressRule{rule}

				// Turn off TLS so we have better control over the number of
				// generated request routing rules.
				ingress.Spec.TLS = nil
				ingresses = append(ingresses, ingress)
				Expect(configBuilder.k8sContext.Caches.Ingress.Add(ingress)).To(Succeed())
			}

			cbCtx = &ConfigBuilderContext{
				IngressList:           ingresses,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
		})

		Context("when listeners are of multi-site type", func() {
			var minPriority, maxPriority int32 = 19000, 19500

			BeforeEach(func() {
				for idx, ingress := range ingresses {
					ingress.Annotations[annotations.HostNameExtensionKey] = fmt.Sprintf("host-%d", idx)
				}

				Expect(configBuilder.Listeners(cbCtx)).To(Succeed())
				Expect(configBuilder.RequestRoutingRules(cbCtx)).To(Succeed())
			})

			It("should have the expected number of request routing rules", func() {
				Expect(*configBuilder.appGw.RequestRoutingRules).To(HaveLen(ruleCount))
			})

			It("should all have unique priorities assigned", func() {
				Expect(*configBuilder.appGw.RequestRoutingRules).To(Satisfy(haveUniquePriorities))
			})

			It("should all have priorities in range", func() {
				Expect(*configBuilder.appGw.RequestRoutingRules).To(HaveEach(Satisfy(func(r n.ApplicationGatewayRequestRoutingRule) bool {
					return minPriority <= *r.Priority && *r.Priority < maxPriority
				})))
			})
		})

		Context("when listeners are of basic type", func() {
			var minPriority, maxPriority int32 = 19500, 20000

			BeforeEach(func() {
				Expect(configBuilder.Listeners(cbCtx)).To(Succeed())
				Expect(configBuilder.RequestRoutingRules(cbCtx)).To(Succeed())
			})

			It("should have the expected number of request routing rules", func() {
				Expect(*configBuilder.appGw.RequestRoutingRules).To(HaveLen(ruleCount))
			})

			It("should all have unique priorities assigned", func() {
				Expect(*configBuilder.appGw.RequestRoutingRules).To(Satisfy(haveUniquePriorities))
			})

			It("should all have priorities in range", func() {
				Expect(*configBuilder.appGw.RequestRoutingRules).To(HaveEach(Satisfy(func(r n.ApplicationGatewayRequestRoutingRule) bool {
					return minPriority <= *r.Priority && *r.Priority < maxPriority
				})))
			})
		})

	})

	Context("test preparePathFromPathType", func() {
		It("should append * when pathType is Prefix", func() {
			pathType := networking.PathTypePrefix

			Expect(preparePathFromPathType("/path", &pathType)).To(Equal("/path*"))
			Expect(preparePathFromPathType("/path*", &pathType)).To(Equal("/path*"))
		})

		It("should trim when pathType is Exact", func() {
			pathType := networking.PathTypeExact

			Expect(preparePathFromPathType("/path", &pathType)).To(Equal("/path"))
			Expect(preparePathFromPathType("/path*", &pathType)).To(Equal("/path"))
		})

		It("should not modify when pathType is ImplementationSpecific", func() {
			pathType := networking.PathTypeImplementationSpecific

			Expect(preparePathFromPathType("/path", &pathType)).To(Equal("/path"))
			Expect(preparePathFromPathType("/path*", &pathType)).To(Equal("/path*"))
		})

		It("should not modify when pathType is nil", func() {
			Expect(preparePathFromPathType("/path", nil)).To(Equal("/path"))
			Expect(preparePathFromPathType("/path*", nil)).To(Equal("/path*"))
		})
	})

	Context("test isPathCatchAll", func() {
		// Application Gateway doesn't allow exact path for "/"
		It("should be false if / and pathType:exact", func() {
			pathTypeExact := networking.PathTypeExact
			Expect(isPathCatchAll("/", &pathTypeExact)).To(BeTrue())
			Expect(isPathCatchAll("/*", &pathTypeExact)).To(BeTrue())
		})

		It("should be true if / and pathType:nil", func() {
			Expect(isPathCatchAll("/", nil)).To(BeTrue())
			Expect(isPathCatchAll("/*", nil)).To(BeTrue())
		})

		It("should be true if / and pathType:prefix", func() {
			pathTypePrefix := networking.PathTypePrefix
			Expect(isPathCatchAll("/", &pathTypePrefix)).To(BeTrue())
			Expect(isPathCatchAll("/*", &pathTypePrefix)).To(BeTrue())
		})

		It("should be true if / and pathType:implementationSpecific", func() {
			pathTypeIMS := networking.PathTypeImplementationSpecific
			Expect(isPathCatchAll("/", &pathTypeIMS)).To(BeTrue())
			Expect(isPathCatchAll("/*", &pathTypeIMS)).To(BeTrue())
		})
	})
})

func haveUniquePriorities(rules []n.ApplicationGatewayRequestRoutingRule) bool {
	priorities := make(map[int32]bool)
	for _, r := range rules {
		if _, ok := priorities[*r.Priority]; ok {
			return false
		}
	}
	return true
}
