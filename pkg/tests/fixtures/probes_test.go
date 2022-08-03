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
	Context("Testing GetApplicationGatewayProbe", func() {
		It("should work as expected", func() {
			host := "host"
			path := "path"
			actual := GetApplicationGatewayProbe(&host, &path)
			expected := "probe-name-aG9zdA-cGF0aA"
			Expect(*actual.Name).To(Equal(expected))
		})
	})
})
