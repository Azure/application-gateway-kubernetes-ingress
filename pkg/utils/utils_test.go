// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build unittest

package utils

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Suite")
}

var _ = Describe("Utils", func() {
	Describe("Testing `utils` helpers", func() {

		Context("Testing the Kubernetes namespace generator", func() {
			It("Given a namespace and resource it should return the Kubernetes resource identifier.", func() {
				Expect(GetResourceKey("default", "pod")).To(Equal("default/pod"))
			})
		})

		Context("Test GetLastChunkOfSlashed", func() {
			It("Should return the last slice of a string split on a slash.", func() {
				Expect(GetLastChunkOfSlashed("a/b/c")).To(Equal("c"))
			})

			It("Should return the full string when there are no slashes.", func() {
				Expect(GetLastChunkOfSlashed("abc")).To(Equal("abc"))
			})
		})

		Context("Test SaveToFile", func() {
			It("should return the path to the temp file and no error", func() {
				pathToFile, err := SaveToFile("blah", []byte("content"))
				Expect(err).ToNot(HaveOccurred())
				Expect(pathToFile).To(ContainSubstring("blah"))
			})
		})

		Context("Test PrettyJSON", func() {
			It("should return pretty JSON and no error", func() {
				prettyJSON, err := PrettyJSON([]byte("{\"name\":\"baba yaga\"}"), "--prefix--")
				Expect(err).ToNot(HaveOccurred())
				Expect(prettyJSON).To(Equal([]byte(`{
--prefix--    "name": "baba yaga"
--prefix--}`)))
			})
		})

		Context("Test GetHashCode", func() {
			It("should generate a deterministic hash", func() {
				hashcode := GetHashCode([]string{"testing hash generation"})
				Expect(hashcode).To(Equal("28a37ff7b783ffb4696dfb7774331163"))
			})
		})

		Context("Test RandStringRunes", func() {
			It("should generate n length string", func() {
				Expect(len(RandStringRunes(10))).To(Equal(10))
			})

			It("should not fail when n = 0", func() {
				Expect(len(RandStringRunes(0))).To(Equal(0))
			})
		})
	})
})
