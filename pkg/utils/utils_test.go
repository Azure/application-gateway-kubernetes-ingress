// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

//go:build unittest
// +build unittest

package utils

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Suite")
}

var _ = Describe("Utils", func() {
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

	DescribeTable("Test RemoveDuplicateStrings",
		func(input []string, expected []string) {
			Expect(RemoveDuplicateStrings(input)).To(Equal(expected))
		},
		Entry(
			"Should remove duplicate strings",
			[]string{"1", "1", "2", "3", "4", "3", "5", "1"},
			[]string{"1", "2", "3", "4", "5"},
		),
		Entry(
			"Should handle slices with no duplicates",
			[]string{"1", "1", "2", "3", "4", "3", "5", "1"},
			[]string{"1", "2", "3", "4", "5"},
		),
		Entry(
			"Should return empty slice if input is empty",
			[]string{},
			[]string{},
		),
		Entry(
			"Should return nil if input is nil",
			[]string(nil),
			[]string(nil),
		),
	)

	Context("Test ParseNamespacedName", func() {
		It("should return namespace and name", func() {
			namespace, name, err := ParseNamespacedName("namespace/name")
			Expect(err).ToNot(HaveOccurred())
			Expect(namespace).To(Equal("namespace"))
			Expect(name).To(Equal("name"))
		})

		It("should return error when namespaced name is invalid", func() {
			_, _, err := ParseNamespacedName("namespace")
			Expect(err).To(HaveOccurred())
		})
	})
})
