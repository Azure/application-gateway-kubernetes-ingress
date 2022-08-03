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
	Context("Testing GetListenerBasic", func() {
		It("should work as expected", func() {
			actual := GetListenerBasic()
			Expect(*actual.Name).To(Equal("HTTPListener-Basic"))
		})
	})

	Context("Testing GetDefaultListener", func() {
		It("should work as expected", func() {
			actual := GetDefaultListener()
			Expect(*actual.Name).To(Equal("fl-80"))
		})
	})

	Context("Testing GetListenerPathBased1", func() {
		It("should work as expected", func() {
			actual := GetListenerPathBased1()
			Expect(*actual.Name).To(Equal("HTTPListener-PathBased"))
		})
	})

	Context("Testing GetListenerPathBased2", func() {
		It("should work as expected", func() {
			actual := GetListenerPathBased2()
			Expect(*actual.Name).To(Equal("HTTPListener-PathBased2"))
		})
	})

	Context("Testing GetListenerUnassociated", func() {
		It("should work as expected", func() {
			actual := GetListenerUnassociated()
			Expect(*actual.Name).To(Equal("HTTPListener-Unassociated"))
		})
	})
})
