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
	Context("Testing GetCertificate1", func() {
		It("should work as expected", func() {
			actual := GetCertificate1()
			Expect(*actual.Name).To(Equal("Certificate-1"))
		})
	})

	Context("Testing GetCertificate2", func() {
		It("should work as expected", func() {
			actual := GetCertificate2()
			Expect(*actual.Name).To(Equal("Certificate-2"))
		})
	})

	Context("Testing GetCertificate3", func() {
		It("should work as expected", func() {
			actual := GetCertificate3()
			Expect(*actual.Name).To(Equal("Certificate-3"))
		})
	})
})
