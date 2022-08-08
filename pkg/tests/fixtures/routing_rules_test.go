// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Fixtures", func() {
	Context("Testing GetRequestRoutingRulePathBased1", func() {
		It("should work as expected", func() {
			actual := GetRequestRoutingRulePathBased1()
			expected := "x/y/z/BackendAddressPool-1"
			Expect(*actual.BackendAddressPool.ID).To(Equal(expected))
		})
	})

	Context("Testing GetRequestRoutingRulePathBased2", func() {
		It("should work as expected", func() {
			actual := GetRequestRoutingRulePathBased2()
			expected := "x/y/z/BackendAddressPool-1"
			Expect(*actual.BackendAddressPool.ID).To(Equal(expected))
		})
	})

	Context("Testing GetRequestRoutingRuleBasic", func() {
		It("should work as expected", func() {
			actual := GetRequestRoutingRuleBasic()
			expected := "x/y/z/BackendAddressPool-2"
			Expect(*actual.BackendAddressPool.ID).To(Equal(expected))
		})
	})

	Context("Testing GetDefaultRoutingRule", func() {
		It("should work as expected", func() {
			actual := GetDefaultRoutingRule()
			expected := "x/y/z/defaultaddresspool"
			Expect(*actual.BackendAddressPool.ID).To(Equal(expected))
		})
	})
})
