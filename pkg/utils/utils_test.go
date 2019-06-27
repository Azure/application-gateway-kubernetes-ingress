package utils_test

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
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

		Context("Test GetLastChunkOfSlashed", func() {
			It("Should return the last slice of a string split on a slash.", func() {
				Expect(utils.GetLastChunkOfSlashed("a/b/c")).To(Equal("c"))
			})

			It("Should return the full string when there are no slashes.", func() {
				Expect(utils.GetLastChunkOfSlashed("abc")).To(Equal("abc"))
			})
		})
	})
})
