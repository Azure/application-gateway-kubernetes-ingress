// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Test the creation of Backend Pools from Ingress definition", func() {
	subset := v1.EndpointSubset{
		Addresses: []v1.EndpointAddress{
			{Hostname: "abc"},
			{IP: "1.1.1.1"},
			{Hostname: "abc"},
			{IP: "1.1.1.1"},
			{Hostname: "xyz"},
			{IP: "2.2.2.2"},
		},
	}

	serviceList := []*v1.Service{
		tests.NewServiceFixture(),
	}

	Context("build a list of BackendAddressPools", func() {
		ing1 := tests.NewIngressFixture()
		ing2 := tests.NewIngressFixture()
		ingressList := []*v1beta1.Ingress{
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
			IngressList: cb.k8sContext.ListHTTPIngresses(),
			ServiceList: serviceList,
		}
		_ = cb.BackendAddressPools(cbCtx)

		It("should contain correct number of backend address pools", func() {
			Expect(len(*cb.appGw.BackendAddressPools)).To(Equal(1))

		})

		It("should contain correct backend address pools", func() {
			Expect(*cb.appGw.BackendAddressPools).To(ContainElement(defaultBackendAddressPool(cb.appGwIdentifier)))
		})
	})

	Context("ensure unique IP addresses", func() {
		ingressList := []*v1beta1.Ingress{tests.NewIngressFixture()}
		cb := newConfigBuilderFixture(nil)
		for _, ingress := range ingressList {
			_ = cb.k8sContext.Caches.Ingress.Add(ingress)
		}
		cbCtx := &ConfigBuilderContext{
			IngressList: cb.k8sContext.ListHTTPIngresses(),
			ServiceList: serviceList,
		}
		_ = cb.BackendAddressPools(cbCtx)
		actualPool := cb.newPool("pool-name", subset)
		It("should contain unique addresses only", func() {
			Expect(len(*actualPool.BackendAddresses)).To(Equal(4))
		})
	})

	Context("ensure correct creation of ApplicationGatewayBackendAddress", func() {
		actual := getAddressesForSubset(subset)
		It("should contain correct number of ApplicationGatewayBackendAddress", func() {
			Expect(len(*actual)).To(Equal(4))
		})
		It("should contain correct set of ordered ApplicationGatewayBackendAddress", func() {
			// The order here is deliberate -- ensure this is properly sorted
			expected := []n.ApplicationGatewayBackendAddress{
				{IPAddress: to.StringPtr("1.1.1.1")},
				{IPAddress: to.StringPtr("2.2.2.2")},
				{Fqdn: to.StringPtr("abc")},
				{Fqdn: to.StringPtr("xyz")},
			}
			Expect(*actual).To(Equal(expected))
		})
	})

	Context("ensure correct creation of ApplicationGatewayBackendAddress", func() {
		ingressList := []*v1beta1.Ingress{tests.NewIngressFixture()}
		cb := newConfigBuilderFixture(nil)
		for _, ingress := range ingressList {
			_ = cb.k8sContext.Caches.Ingress.Add(ingress)
		}
		cbCtx := &ConfigBuilderContext{
			ServiceList: serviceList,
			IngressList: cb.k8sContext.ListHTTPIngresses(),
		}
		_ = cb.BackendAddressPools(cbCtx)

		endpoints := tests.NewEndpointsFixture()
		_ = cb.k8sContext.Caches.Endpoints.Add(endpoints)

		// TODO(draychev): Move to test fixtures
		backendID := backendIdentifier{
			serviceIdentifier: serviceIdentifier{
				Namespace: tests.Namespace,
				Name:      tests.ServiceName,
			},
			Backend: tests.NewIngressBackendFixture(tests.ServiceName, int32(4321)),
			Ingress: tests.NewIngressFixture(),
		}
		serviceBackendPair := serviceBackendPortPair{
			// TODO(draychev): Move to test fixtures
			ServicePort: Port(4321),
			BackendPort: Port(tests.ContainerPort),
		}

		pool := tests.GetApplicationGatewayBackendAddressPool()
		addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
			*pool.Name: pool,
		}

		// -- Action --
		actual := cb.getBackendAddressPool(backendID, serviceBackendPair, addressPools)

		It("should have constructed correct ApplicationGatewayBackendAddressPool", func() {
			// The order here is deliberate -- ensure this is properly sorted
			expectedPoolName := "pool-" + tests.Namespace + "-" + tests.ServiceName + "-4321-bp-9876"
			expected := n.ApplicationGatewayBackendAddressPool{
				Name: to.StringPtr(expectedPoolName),
				ID:   to.StringPtr(cb.appGwIdentifier.addressPoolID(expectedPoolName)),
				Etag: to.StringPtr("*"),
				ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
					BackendIPConfigurations: nil,
					BackendAddresses: &[]n.ApplicationGatewayBackendAddress{
						{
							Fqdn:      nil,
							IPAddress: to.StringPtr("10.9.8.7"),
						},
					},
					ProvisioningState: nil,
				},
			}
			Expect(*actual).To(Equal(expected))
		})
	})

	Context("Test Istio components", func() {
		cb := newConfigBuilderFixture(nil)
		istioDest := istioDestinationIdentifier{}
		istioGateways := []*v1alpha3.Gateway{}
		istioVirtualServices := []*v1alpha3.VirtualService{}

		destinationID := istioDestinationIdentifier{}
		serviceBackendPair := serviceBackendPortPair{}
		addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{}

		It("Should resolve Istio Port names", func() {
			expected := map[Port]interface{}{}
			actual := cb.resolveIstioPortName("portName", &istioDest)
			Expect(actual).To(Equal(expected))
		})

		It("Should get listener config from istio", func() {
			actual := cb.getListenerConfigsFromIstio(istioGateways, istioVirtualServices)
			expected := map[listenerIdentifier]listenerAzConfig{
				listenerIdentifier{FrontendPort: 80, HostName: "", UsePrivateIP: false}: {
					Protocol:                     "Http",
					Secret:                       secretIdentifier{Namespace: "", Name: ""},
					SslRedirectConfigurationName: "",
				},
			}
			Expect(actual).To(Equal(expected))
		})

		It("Should get backend pools from istio", func() {
			actual := cb.getIstioBackendAddressPool(destinationID, serviceBackendPair, addressPools)
			Expect(actual).To(BeNil())
		})

		It("Should get path maps from istio", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: cb.k8sContext.ListHTTPIngresses(),
				ServiceList: serviceList,
			}
			actual := cb.getIstioPathMaps(cbCtx)
			expected := map[listenerIdentifier]*n.ApplicationGatewayURLPathMap{

				listenerIdentifier{FrontendPort: 80, HostName: "", UsePrivateIP: false}: {

					ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
						DefaultBackendAddressPool: &n.SubResource{
							ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
								"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
								"/backendAddressPools/defaultaddresspool"),
						},
						DefaultBackendHTTPSettings: &n.SubResource{
							ID: to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--" +
								"/providers/Microsoft.Network/applicationGateways/--app-gw-name--" +
								"/backendHttpSettingsCollection/defaulthttpsetting"),
						},
						DefaultRewriteRuleSet:        nil,
						DefaultRedirectConfiguration: nil,
						PathRules:                    &[]n.ApplicationGatewayPathRule{},
						ProvisioningState:            nil,
					},
					Name: to.StringPtr("url-80"),
					Etag: to.StringPtr("*"),
					Type: nil,
					ID:   nil,
				},
			}

			Expect(actual).To(Equal(expected))
		})

		It("Should get destinations from istio", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: cb.k8sContext.ListHTTPIngresses(),
				ServiceList: serviceList,
			}
			expectedSettingsList := []n.ApplicationGatewayBackendHTTPSettings{}
			expectedSettinsgPerDestination := map[istioDestinationIdentifier]*n.ApplicationGatewayBackendHTTPSettings{}
			expectedPortsPerDestination := map[istioDestinationIdentifier]serviceBackendPortPair{}

			settingsList, settingsPerDestination, portsPerDestination, err := cb.getIstioDestinationsAndSettingsMap(cbCtx)

			Expect(expectedSettingsList).To(Equal(settingsList))
			Expect(expectedSettinsgPerDestination).To(Equal(settingsPerDestination))
			Expect(expectedPortsPerDestination).To(Equal(portsPerDestination))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
