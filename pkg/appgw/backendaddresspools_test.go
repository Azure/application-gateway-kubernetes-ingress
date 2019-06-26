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
			IngressList: cb.k8sContext.GetHTTPIngressList(),
			ServiceList: serviceList,
		}
		_ = cb.BackendAddressPools(cbCtx)

		It("should contain correct number of backend address pools", func() {
			Expect(len(*cb.appGw.BackendAddressPools)).To(Equal(1))

		})

		It("should contain correct backend address pools", func() {
			props := &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
				BackendIPConfigurations: nil,
				BackendAddresses:        &[]n.ApplicationGatewayBackendAddress{},
				ProvisioningState:       nil,
			}
			expected := n.ApplicationGatewayBackendAddressPool{
				Name: to.StringPtr("defaultaddresspool"),
				Etag: nil,
				Type: nil,
				ID:   nil,
				ApplicationGatewayBackendAddressPoolPropertiesFormat: props,
			}
			Expect(*cb.appGw.BackendAddressPools).To(ContainElement(expected))
		})
	})

	Context("ensure unique IP addresses", func() {
		ingressList := []*v1beta1.Ingress{tests.NewIngressFixture()}
		cb := newConfigBuilderFixture(nil)
		for _, ingress := range ingressList {
			_ = cb.k8sContext.Caches.Ingress.Add(ingress)
		}
		cbCtx := &ConfigBuilderContext{
			IngressList: cb.k8sContext.GetHTTPIngressList(),
			ServiceList: serviceList,
		}
		_ = cb.BackendAddressPools(cbCtx)
		actualPool := newPool("pool-name", subset)
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
			IngressList: cb.k8sContext.GetHTTPIngressList(),
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
			ServicePort: int32(4321),
			BackendPort: tests.ContainerPort,
		}

		pool := tests.GetApplicationGatewayBackendAddressPool()
		addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
			*pool.Name: pool,
		}

		// -- Action --
		actual := cb.getBackendAddressPool(backendID, serviceBackendPair, addressPools)

		It("should have constructed correct ApplicationGatewayBackendAddressPool", func() {
			// The order here is deliberate -- ensure this is properly sorted
			expected := n.ApplicationGatewayBackendAddressPool{
				Name: to.StringPtr("pool-" + tests.Namespace + "-" + tests.ServiceName + "-4321-bp-9876"),
				ID:   nil,
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
})
