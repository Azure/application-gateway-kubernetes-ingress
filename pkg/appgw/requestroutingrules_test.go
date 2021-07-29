// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"

	appgwldp "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/loaddistributionpolicy/v1beta1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test routing rules generations", func() {
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
		sharedListenerID := generateListenerID(ingressPathBased1, rule, n.HTTPS, nil, false)
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
						backendID := generateBackendID(ingress, &rule, &path, &path.Backend, path.Backend.Service.Name)
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
		sharedListenerID := generateListenerID(ingressPathBased, rule, n.HTTPS, nil, false)
		generatedPathMap := pathMaps[sharedListenerID]
		backendIDBasic := generateBackendID(ingressBasic, &ruleBasic, pathBasic, backendBasic, backendBasic.Service.Name)
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
					backendID := generateBackendID(ingressPathBased, &rule, &path, &path.Backend, path.Backend.Service.Name)
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
			listenerID := generateListenerID(ingress, rule, n.HTTP, nil, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(80)))
		})

		It("frontend port is default to 80 when no annotation", func() {
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.HTTP, &overrideFrontendPort, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(80)))
		})

		It("frontend port is default to 443 when https", func() {
			listenerID := generateListenerID(ingress, rule, n.HTTPS, nil, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(443)))
		})

		It("frontend port is default to 443 when https with no annotation", func() {
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.HTTPS, &overrideFrontendPort, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(443)))
		})

		It("frontend port is overridden in annotation", func() {
			ingress.Annotations[annotations.OverrideFrontendPortKey] = "777"
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.HTTP, &overrideFrontendPort, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(777)))
		})

		It("frontend port is out of range", func() {
			ingress.Annotations[annotations.OverrideFrontendPortKey] = "65000"
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.HTTP, &overrideFrontendPort, false)
			Expect(listenerID.FrontendPort).To(Equal(Port(80)))
		})

		It("frontend port is out of range", func() {
			ingress.Annotations[annotations.OverrideFrontendPortKey] = "0"
			overrideFrontendPortFromAnnotation, _ := annotations.OverrideFrontendPort(ingress)
			overrideFrontendPort := Port(overrideFrontendPortFromAnnotation)
			listenerID := generateListenerID(ingress, rule, n.HTTP, &overrideFrontendPort, false)
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
		ingressPathBased.Annotations[annotations.SslRedirectKey] = "false"
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
		sharedListenerID := generateListenerID(ingressPathBased, rule, n.HTTPS, nil, false)
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
					backendID := generateBackendID(ingressPathBased, &rule, &path, &path.Backend, path.Backend.Service.Name)
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
		ingressPathBased.Annotations[annotations.SslRedirectKey] = "false"
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
		sharedListenerID := generateListenerID(ingressPathBased, rule, n.HTTPS, nil, false)
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
					backendID := generateBackendID(ingressPathBased, &rule, &path, &path.Backend, path.Backend.Service.Name)
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
		sharedListenerID := generateListenerID(ingressPathBased1, rule, n.HTTPS, nil, false)
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
					backendID := generateBackendID(ingress, &rule, &path, &path.Backend, path.Backend.Service.Name)
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
					backendID := generateBackendID(ingress, &rule, &path, &path.Backend, path.Backend.Service.Name)
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

	Context("test path-based rule with 1 ingress with 1 path with LDP backend", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ldpTargets := []appgwldp.Target{
			{
				Role:   "active",
				Weight: 10,
				Backend: appgwldp.Backend{
					Service: &networking.IngressServiceBackend{
						Name: service.Name,
						Port: networking.ServiceBackendPort{
							Number: 80,
						},
					},
				},
			},
		}
		ldp := tests.NewLoadDistrbutionPolicyFixture(ldpTargets)
		ingressPathBased := tests.NewIngressWithLoadDistributionPolicyFixture()
		ingressPathBased.Annotations[annotations.SslRedirectKey] = "false"
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased)
		_ = configBuilder.k8sContext.Caches.LoadDistributionPolicy.Add(ldp)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingressPathBased},
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
		It("has default backend pool", func() {
			Expect(generatedPathMap.DefaultBackendAddressPool).To(Not(BeNil()))
		})
		It("has default backend http settings", func() {
			Expect(generatedPathMap.DefaultBackendHTTPSettings).To(Not(BeNil()))
		})
		It("should have one path rules", func() {
			checkPathRules(generatedPathMap, 1)
		})
		It("should be able to merge all the path rules into the same path map", func() {
			ingress := cbCtx.IngressList[0]
			for ruleIdx, rule := range ingress.Spec.Rules {
				for pathIdx, path := range rule.HTTP.Paths {
					backendID := generateBackendID(ingress, &rule, &path, &path.Backend, service.Name)
					ldpResourceName := generateLoadDistributionName(ingress.Namespace, path.Backend.Resource.Name)
					loadDistributionPolicyID := configBuilder.appGwIdentifier.LoadDistributionPolicyID(ldpResourceName)
					httpSettingID := configBuilder.appGwIdentifier.HTTPSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), serviceBackendPortToStr(ldp.Spec.Targets[0].Backend.Service.Port), Port(tests.ContainerPort), backendID.Ingress.Name))
					pathRuleName := generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, ruleIdx, pathIdx)
					expectedPathRule := n.ApplicationGatewayPathRule{
						Name: to.StringPtr(pathRuleName),
						Etag: to.StringPtr("*"),
						ID:   to.StringPtr(configBuilder.appGwIdentifier.pathRuleID(*generatedPathMap.Name, pathRuleName)),
						ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
							Paths: &[]string{
								path.Path,
							},
							LoadDistributionPolicy: &n.SubResource{ID: to.StringPtr(loadDistributionPolicyID)},
							BackendHTTPSettings:    &n.SubResource{ID: to.StringPtr(httpSettingID)},
						},
					}
					Expect(*generatedPathMap.PathRules).To(ContainElement(expectedPathRule))
				}
			}
		})
	})

	Context("test path-based rule with 1 ingress with 1 path with non existent LDP backend", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingressPathBased := tests.NewIngressWithLoadDistributionPolicyFixture()
		ingressPathBased.Annotations[annotations.SslRedirectKey] = "false"
		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingressPathBased)

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingressPathBased},
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
		It("has default backend pool", func() {
			Expect(generatedPathMap.DefaultBackendAddressPool).To(Not(BeNil()))
		})
		It("has default backend http settings", func() {
			Expect(generatedPathMap.DefaultBackendHTTPSettings).To(Not(BeNil()))
		})
		It("should not generate any rules", func() {
			Expect(generatedPathMap.PathRules).To(BeNil())
		})
	})
})
