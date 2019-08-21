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
	Context("Testing GetAzureIngressProhibitedTargets", func() {
		It("should work as expected", func() {
			actual := GetAzureIngressProhibitedTargets()
			expected := "bye.com"
			Expect(actual[0].Spec.Hostname).To(Equal(expected))
		})
	})
})
