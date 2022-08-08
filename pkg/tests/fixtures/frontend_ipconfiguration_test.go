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
	Context("Testing GetPublicIPConfiguration", func() {
		It("should work as expected", func() {
			actual := GetPublicIPConfiguration()
			Expect(*actual.Name).To(Equal("PublicIP"))
		})
	})

	Context("Testing GetPrivateIPConfiguration", func() {
		It("should work as expected", func() {
			actual := GetPrivateIPConfiguration()
			Expect(*actual.Name).To(Equal("PrivateIP"))
		})
	})
})
