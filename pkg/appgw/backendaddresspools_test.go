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
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Test the creation of Backend Pools from Ingress definition", func() {

	Context("ingress rules without certificates", func() {
		cb := newConfigBuilderFixture(nil)
		actualPools := cb.getPools()

		It("should contain correct number of backend address pools", func() {
			Expect(len(actualPools)).To(Equal(2))

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
			Expect(actualPools).To(ContainElement(expected))
		})
	})
})
