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
	Context("Testing GetHTTPSettings1", func() {
		It("should work as expected", func() {
			actual := GetHTTPSettings1()
			Expect(*actual.Name).To(Equal("BackendHTTPSettings-1"))
		})
	})

	Context("Testing GetHTTPSettings2", func() {
		It("should work as expected", func() {
			actual := GetHTTPSettings2()
			Expect(*actual.Name).To(Equal("BackendHTTPSettings-2"))
		})
	})

	Context("Testing GetHTTPSettings3", func() {
		It("should work as expected", func() {
			actual := GetHTTPSettings3()
			Expect(*actual.Name).To(Equal("BackendHTTPSettings-3"))
		})
	})
})
