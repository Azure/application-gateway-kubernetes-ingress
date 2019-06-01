// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package tests

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestIngressTestFixtureFactories(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ingress Test Fixture Factories Suite")
}

var _ = Describe("Test Fixture Object Factories", func() {

	Context("Ensure we can create test fixture objects", func() {
		It("Should create a simple Ingress", func() {
			actual, err := GetIngress()
			Expect(err).To(BeNil())
			Expect(actual).To(Not(BeNil()))
			Expect(actual.Name).To(Equal("websocket-ingress"))
			Expect(len(actual.Spec.Rules)).To(Equal(1))
		})

		It("Should create a complex Ingress", func() {
			actual, err := GetIngressComplex()
			Expect(err).To(BeNil())
			Expect(actual).To(Not(BeNil()))
			Expect(actual.Name).To(Equal("websocket-ingress"))
			Expect(len(actual.Spec.Rules)).To(Equal(3))
		})

		It("Should create 2 Ingresses in separate namespaces", func() {
			ingresses, err := GetIngressNamespaced()
			Expect(err).To(BeNil())
			Expect(ingresses).To(Not(BeNil()))

			names := []string{(*ingresses)[0].Name, (*ingresses)[1].Name}
			Expect(names).To(ContainElement("ingress-coffeeshop"))
			Expect(names).To(ContainElement("ingress-roastery"))

			namespaces := []string{(*ingresses)[0].Namespace, (*ingresses)[1].Namespace}
			Expect(namespaces).To(ContainElement("factory-ns"))
			Expect(namespaces).To(ContainElement("store-ns"))
		})
	})
})
