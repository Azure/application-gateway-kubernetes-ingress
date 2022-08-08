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
	Context("Testing GetAppGateway", func() {
		It("should work as expected", func() {
			actual := GetAppGateway()
			expected := "Certificate-1"
			Expect(*(*(actual.SslCertificates))[0].Name).To(Equal(expected))
		})
	})
})
