// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestIt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Run All main.go Tests")
}

var _ = Describe("Test functions used in main.go", func() {

	Context("test namespace env var parser", func() {
		It("should parse comma separated namespaces from env var", func() {
			actual := getNamespacesToWatch("")
			expected := []string{}
			Expect(actual).To(Equal(expected))
		})
		It("should parse comma separated namespaces from env var", func() {
			actual := getNamespacesToWatch("singleNamespace")
			expected := []string{"singleNamespace"}
			Expect(actual).To(Equal(expected))
		})
		It("should parse comma separated namespaces from env var", func() {
			actual := getNamespacesToWatch("two,one")
			expected := []string{"one", "two"}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("test getVerbosity", func() {
		flagVerbosity := 9
		envVerbosity := "8"
		It("should return verbosity level integer", func() {
			actual := getVerbosity(flagVerbosity, envVerbosity)
			Expect(actual).To(Equal(8))
		})
		It("should return verbosity level integer", func() {
			envVerbosity := ""
			actual := getVerbosity(flagVerbosity, envVerbosity)
			Expect(actual).To(Equal(9))
		})
	})

})
