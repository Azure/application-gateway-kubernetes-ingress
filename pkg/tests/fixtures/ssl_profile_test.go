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
	Context("Testing GetSslProfile1", func() {
		It("should work as expected", func() {
			actual := GetSslProfile1()
			Expect(*actual.Name).To(Equal("hardend-tls"))
		})
	})
})
