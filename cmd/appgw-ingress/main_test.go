// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"testing"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
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
			actual := validateNamespaces([]string{}, &kubernetes.Clientset{})
			Ω(actual).Should(Succeed())
		})
	})

	Context("test getNamespacesToWatch", func() {
		It("should return a single namespace to watch", func() {
			actual := getNamespacesToWatch("some-env-var")
			Ω(actual).Should(Equal([]string{"some-env-var"}))
		})
		It("should return a list of namespaces to watch", func() {
			actual := getNamespacesToWatch("a,b,c")
			Ω(actual).Should(Equal([]string{"a", "b", "c"}))
		})
		It("should return empty list of namespaces to watch", func() {
			actual := getNamespacesToWatch("")
			Ω(actual).Should(Equal([]string{}))
		})
	})

	Context("test waitForAzureAuth", func() {
		client := n.ApplicationGatewaysClient{}
		It("should try and panic", func() {
			env := environment.EnvVariables{}
			fn := func() {
				_ = waitForAzureAuth(env, &client, 0)
			}
			Ω(fn).Should(Panic())
		})
	})
})
