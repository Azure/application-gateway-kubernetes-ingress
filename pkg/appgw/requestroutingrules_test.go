// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test routing rules generations", func() {
	Context("test path-based rule with ingress", func() {
		configBuilder := newConfigBuilderFixture(nil)
		endpoint := tests.NewEndpointsFixture()
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		ingress1 := tests.NewIngressFixture()
		ingress1.Annotations[annotations.SslRedirectKey] = "false"
		ingress2 := tests.NewIngressFixture()
		ingress2.Name = "ingress1"
		ingress2.Annotations[annotations.SslRedirectKey] = "false"
		newBackend := tests.NewIngressBackendFixture(tests.ServiceName, 80)
		newRule := tests.NewIngressRuleFixture(tests.Host, tests.URLPath3, *newBackend)
		ingress2.Spec.Rules = []v1beta1.IngressRule{
			newRule,
		}

		_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
		_ = configBuilder.k8sContext.Caches.Service.Add(service)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress1)
		_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress2)

		cbCtx := &ConfigBuilderContext{
			IngressList: []*v1beta1.Ingress{ingress1, ingress2},
			ServiceList: []*v1.Service{service},
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)
		_ = configBuilder.RequestRoutingRules(cbCtx)

		rule := &ingress1.Spec.Rules[0]

		_ = configBuilder.Listeners(cbCtx)
		// !! Action !! -- will mutate pathMap struct
		pathMaps := configBuilder.getPathMaps(cbCtx)
		sharedListenerID := generateListenerID(rule, n.HTTPS, nil, false)
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
		It("should have correct path rule for /api1 from ingress", func() {
			for _, ingress := range cbCtx.IngressList {
				for _, rule := range ingress.Spec.Rules {
					for _, path := range rule.HTTP.Paths {
						backendID := generateBackendID(ingress, &rule, &path, &path.Backend)
						backendPoolID := configBuilder.appGwIdentifier.addressPoolID(generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), tests.ContainerPort))
						httpSettingID := configBuilder.appGwIdentifier.httpSettingsID(generateHTTPSettingsName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), tests.ContainerPort, backendID.Ingress.Name))
						expectedPathRule := n.ApplicationGatewayPathRule{
							Name: to.StringPtr(generatePathRuleName(backendID.Ingress.Namespace, backendID.Ingress.Name, "0")),
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
		listenerID := generateListenerID(rule, n.HTTP, nil, false)
		It("has no default backend pool", func() {
			Expect(pathMap[listenerID].DefaultBackendAddressPool).To(BeNil())
		})
		It("has no default backend http settings", func() {
			Expect(pathMap[listenerID].DefaultBackendHTTPSettings).To(BeNil())
		})

		expectedRedirectID := configBuilder.appGwIdentifier.redirectConfigurationID(
			generateSSLRedirectConfigurationName(listenerIdentifier{
				HostName:     rule.Host,
				FrontendPort: 443,
			}))
		actualID := *(pathMap[listenerID].DefaultRedirectConfiguration.ID)
		It("generated expected ID", func() {
			Expect(actualID).To(Equal(expectedRedirectID))
		})
		It("should still have 2 path rules", func() {
			Expect(2).To(Equal(len(*pathMap[listenerID].PathRules)))
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
			IngressList: []*v1beta1.Ingress{ingress},
			ServiceList: []*v1.Service{service},
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
		_ = configBuilder.Listeners(cbCtx)

		// !! Action !! -- will mutate pathMap struct
		pathMap := configBuilder.getPathMaps(cbCtx)

		rule := &ingress.Spec.Rules[0]
		listenerID := generateListenerID(rule, n.HTTP, nil, false)
		It("has no default backend pool", func() {
			Expect(pathMap[listenerID].DefaultBackendAddressPool).To(BeNil())
		})
		It("has no default backend http settings", func() {
			Expect(pathMap[listenerID].DefaultBackendHTTPSettings).To(BeNil())
		})
		It("has no pathrules", func() {
			Expect(pathMap[listenerID].PathRules).To(BeNil())
		})

		expectedRedirectID := configBuilder.appGwIdentifier.redirectConfigurationID(
			generateSSLRedirectConfigurationName(listenerIdentifier{
				HostName:     rule.Host,
				FrontendPort: 443,
			}))
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
			IngressList: []*v1beta1.Ingress{ingress},
			ServiceList: []*v1.Service{service},
		}

		_ = configBuilder.BackendHTTPSettingsCollection(cbCtx)
		_ = configBuilder.BackendAddressPools(cbCtx)
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
							"/backendAddressPools/pool---namespace-----service-name---80-bp-9876"),
					},
					BackendHTTPSettings: &n.SubResource{
						ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
							"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
							"/backendHttpSettingsCollection/bp---namespace-----service-name---80-9876---name--"),
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
