package utils_test

import (
	utils "github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("Testing `UnorderedSets`", func() {
		var testSet utils.UnorderedSet
		BeforeEach(func() {
			testSet = utils.NewUnorderedSet()
			testSet.Insert(1)
			testSet.Insert(2)
			testSet.Insert(3)
			testSet.Insert(4)
			testSet.Insert(4)
		})

		Context("Inserting non-unique elements", func() {
			It("Should only store unique elements", func() {
				Expect(testSet.Size()).To(Equal(4))
			})
		})

		Context("Erasing an element", func() {
			It("Should remove the element", func() {
				testSet.Erase(4)
				Expect(testSet.Contains(4)).To(BeFalse())
				Expect(testSet.Size()).To(Equal(3))
			})
		})

		Context("Clearing the unordered set", func() {
			It("Should erase all elements", func() {
				testSet.Clear()
				Expect(testSet.Size()).To(Equal(0))
			})
		})

	})

	Describe("Testing `utils` helpers", func() {
		Context("Testing integer comparators", func() {
			It("Should return maximum of two 64-bit integers", func() {
				Expect(utils.MaxInt64(int64(101), int64(100))).To(Equal(int64(101)))
				Expect(utils.MaxInt64(int64(100), int64(101))).To(Equal(int64(101)))
			})

			It("Should return maximum of two 32-bit integers", func() {
				Expect(utils.MaxInt32(int32(101), int32(100))).To(Equal(int32(101)))
				Expect(utils.MaxInt32(int32(100), int32(101))).To(Equal(int32(101)))
			})
		})

		Context("Testing string helpers", func() {
			It("Should return a string, which is a formatted list of integers", func() {
				Expect(utils.IntsToString([]int{1, 2, 3, 4, 5, 6}, ";")).To(Equal("1;2;3;4;5;6"))
			})
		})

		Context("Testing the Kubernetes namespace generator", func() {
			It("Given a namespace and resource it should return the Kubernetes resource identifier.", func() {
				Expect(utils.GetResourceKey("default", "pod")).To(Equal("default/pod"))
			})
		})
	})
})
