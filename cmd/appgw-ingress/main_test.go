// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
)

func TestIt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Run All main.go Tests")
}

var _ = Describe("Test functions used in main.go", func() {

	Context("test namespace env var parser", func() {
		It("should parse comma separated namespaces from env var", func() {
			actual := getNamespacesToWatch("")
			Expect(actual).To(BeNil())
		})
		It("should parse comma separated namespaces from env var", func() {
			actual := getNamespacesToWatch("singleNamespace")
			expected := &map[string]interface{}{"singleNamespace": nil}
			Expect(actual).To(Equal(expected))
		})
		It("should parse comma separated namespaces from env var", func() {
			actual := getNamespacesToWatch("two,one")
			expected := &map[string]interface{}{"one": nil, "two": nil}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("test getVerbosity", func() {
		flagVerbosity := 9
		envVerbosity := "8"
		It("should return verbosity level based on an environment variable", func() {
			actual := getVerbosity(flagVerbosity, envVerbosity)
			Expect(actual).To(Equal(8))
		})
		It("should return verbosity level based on a command line flag", func() {
			envVerbosity := ""
			actual := getVerbosity(flagVerbosity, envVerbosity)
			Expect(actual).To(Equal(9))
		})
	})

	Context("test validateNamespaces", func() {
		It("should validate the namespaces", func() {
			actual := validateNamespaces(&map[string]interface{}{}, &kubernetes.Clientset{})
			Ω(actual).Should(Succeed())
		})
	})

	Context("test getNamespacesToWatch", func() {
		It("should return a single namespace to watch", func() {
			actual := getNamespacesToWatch("some-env-var")
			Ω(actual).Should(Equal(&map[string]interface{}{"some-env-var": nil}))
		})
		It("should return a list of namespaces to watch", func() {
			actual := getNamespacesToWatch("a,b,c")
			Ω(actual).Should(Equal(&map[string]interface{}{"a": nil, "b": nil, "c": nil}))
		})
		It("should return empty list of namespaces to watch", func() {
			actual := getNamespacesToWatch("")
			Ω(actual).To(BeNil())
		})
	})
})
