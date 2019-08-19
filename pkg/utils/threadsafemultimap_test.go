package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test utils.ThreadSafeMultiMap", func() {
	Describe("Test it", func() {
		Context("the whole thing", func() {
			tsmm := NewThreadsafeMultimap()
			It("", func() {
				tsmm.Insert("name", "baba yaga")
				tsmm.Insert("name", "baba yaga")
				tsmm.Insert("name", "ursula")
				tsmm.Insert("nick", "ursula")
				tsmm.Insert("age", 321)
				Expect(tsmm.ContainsValue("baba yaga")).To(BeTrue())
				Expect(tsmm.ContainsPair("name", "baba yaga")).To(BeTrue())
				Expect(tsmm.ContainsPair("name", "ursula")).To(BeTrue())
				Expect(tsmm.ContainsPair("age", 321)).To(BeTrue())

				actual := tsmm.Erase("name")
				Expect(actual).To(BeTrue())
				actualAgain := tsmm.Erase("name")
				Expect(actualAgain).To(BeFalse())
				Expect(tsmm.ContainsPair("name", "baba yaga")).To(BeFalse())
				Expect(tsmm.ContainsPair("name", "ursula")).To(BeFalse())
				Expect(tsmm.ContainsPair("age", 321)).To(BeTrue())
				tsmm.Clear("name")
				Expect(tsmm.ContainsPair("name", "ursula")).To(BeFalse())
				Expect(tsmm.ContainsPair("age", 321)).To(BeTrue())
				Expect(tsmm.ContainsValue("baba yaga")).To(BeFalse())
				Expect(tsmm.ContainsValue("ursula")).To(BeTrue())
			})
		})

	})
})
