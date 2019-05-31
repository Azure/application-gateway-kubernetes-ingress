// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package utils

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUnorderedSet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Custom Matcher Suite")
}

var _ = Describe("UnorderedSet", func() {

	Context("when seeded with {'one', 'two'} and mutated", func() {

		// RegisterFailHandler(ginkgo.Fail)
		// defer ginkgo.GinkgoRecover()

		set := NewUnorderedSet()
		set.Insert("one")
		set.Insert("two")
		set.Insert("one")
		set.Insert("three")
		set.Erase("three")

		It("should succeed", func() {
			Expect(set.Size()).To(Equal(2))
			Expect(set.Contains("one")).To(Equal(true))
			Expect(set.Contains("two")).To(Equal(true))
			Expect(set.Contains("three")).To(Equal(false))
		})
	})
})
