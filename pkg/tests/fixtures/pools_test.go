// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Fixtures", func() {
	Context("Testing GetDefaultBackendPool", func() {
		It("should work as expected", func() {
			actual := GetDefaultBackendPool()
			Expect(*actual.Name).To(Equal("defaultaddresspool"))
		})
	})

	Context("Testing GetBackendPool1", func() {
		It("should work as expected", func() {
			actual := GetBackendPool1()
			Expect(*actual.Name).To(Equal("BackendAddressPool-1"))
		})
	})

	Context("Testing GetBackendPool2", func() {
		It("should work as expected", func() {
			actual := GetBackendPool2()
			Expect(*actual.Name).To(Equal("BackendAddressPool-2"))
		})
	})

	Context("Testing GetBackendPool3", func() {
		It("should work as expected", func() {
			actual := GetBackendPool3()
			Expect(*actual.Name).To(Equal("BackendAddressPool-3"))
		})
	})
})
