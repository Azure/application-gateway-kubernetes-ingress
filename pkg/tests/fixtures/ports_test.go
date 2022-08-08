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
	Context("Testing GetDefaultPort", func() {
		It("should work as expected", func() {
			actual := GetDefaultPort()
			expected := "fp-80"
			Expect(*actual.Name).To(Equal(expected))
		})
	})
})
