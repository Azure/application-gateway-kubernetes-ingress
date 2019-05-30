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
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Test the creation of Backend Pools from Ingress definition", func() {

	Context("build a list of BackendAddressPools", func() {
		ing1 := newIngressFixture()
		ing2 := newIngressFixture()
		ingressList := []*v1beta1.Ingress{
			ing1,
			ing2,
		}
		cb := newConfigBuilderFixture(nil)
		_, _ = cb.BackendAddressPools(ingressList)

		It("should contain correct number of backend address pools", func() {
			Expect(len(*cb.appGwConfig.BackendAddressPools)).To(Equal(1))

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
			Expect(*cb.appGwConfig.BackendAddressPools).To(ContainElement(expected))
		})
	})

	Context("ensure unique IP addresses", func() {
		ingressList := []*v1beta1.Ingress{newIngressFixture()}
		cb := newConfigBuilderFixture(nil)
		_, _ = cb.BackendAddressPools(ingressList)
		subset := v1.EndpointSubset{
			Addresses: []v1.EndpointAddress{
				{IP: "1.1.1.1"},
				{IP: "1.1.1.1"},
				{IP: "2.2.2.2"},
				{Hostname: "abc"},
				{Hostname: "abc"},
				{Hostname: "xyz"},
			},
		}
		actualPool := newPool("pool-name", subset)
		It("should contain unique addresses only", func() {
			Expect(len(*actualPool.BackendAddresses)).To(Equal(4))
		})
	})
})
