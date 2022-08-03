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
	Context("Testing GetURLPathMap1", func() {
		It("should work as expected", func() {
			actual := GetURLPathMap1()
			Expect(*actual.Name).To(Equal("URLPathMap-1"))
		})
	})

	Context("Testing GetPathRulePathBased1", func() {
		It("should work as expected", func() {
			actual := GetPathRulePathBased1()
			Expect(*actual.Name).To(Equal("PathRule-1URLPathMap-1"))
		})
	})

	Context("Testing GetPathRuleBasic", func() {
		It("should work as expected", func() {
			actual := GetPathRuleBasic()
			Expect(*actual.Name).To(Equal("PathRule-Basic"))
		})
	})

	Context("Testing GetDefaultURLPathMap", func() {
		It("should work as expected", func() {
			actual := GetDefaultURLPathMap()
			Expect(*actual.Name).To(Equal("default-pathmap-name"))
		})
	})

	Context("Testing GetURLPathMap2", func() {
		It("should work as expected", func() {
			actual := GetURLPathMap2()
			Expect(*actual.Name).To(Equal("URLPathMap-2"))
		})
	})

	Context("Testing GetPathRulePathBased2", func() {
		It("should work as expected", func() {
			actual := GetPathRulePathBased2()
			Expect(*actual.Name).To(Equal("PathRule-1URLPathMap-2"))
		})
	})
})
