// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build unittest

package tests

import (
	"testing"

	networking "k8s.io/api/networking/v1"

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

	Context("Test GetIngress", func() {
		It("should work", func() {
			actual, err := GetIngress()
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Spec.Rules[0].Host).To(Equal("ws.contoso.com"))
		})
	})

	Context("Test GetIngressComplex", func() {
		It("should work", func() {
			actual, err := GetIngressComplex()
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Spec.Rules[0].Host).To(Equal("ws.contoso.com"))
		})
	})

	Context("Test GetIngressNamespaced", func() {
		It("should work", func() {
			actual, err := GetIngressNamespaced()
			Expect(err).ToNot(HaveOccurred())
			Expect((*actual)[0].Spec.Rules[0].Host).To(Equal("cafe.contoso.com"))
		})
	})

	Context("Test getIngress", func() {
		It("should throw an error because the file does not exist", func() {
			_, err := getIngress("blahBlahBlah")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test GetApplicationGatewayBackendAddressPool", func() {
		It("should work", func() {
			actual := GetApplicationGatewayBackendAddressPool()
			Expect(*actual.Name).To(Equal("defaultaddresspool"))
		})
	})

	Context("Test NewIngressBackendFixture", func() {
		It("should work", func() {
			actual := NewIngressBackendFixture("service-name", int32(123))
			Expect(actual.Service.Name).To(Equal("service-name"))
		})
	})

	Context("Test NewIngressRuleFixture", func() {
		It("should work", func() {
			actual := NewIngressRuleFixture("host", "urlPath", networking.IngressBackend{})
			Expect(actual.Host).To(Equal("host"))
		})
	})

	Context("Test NewIngressFixture", func() {
		It("should work", func() {
			actual := NewIngressFixture()
			Expect(actual.Name).To(Equal("--name--"))
		})
	})

	Context("Test NewServicePortsFixture", func() {
		It("should work", func() {
			actual := NewServicePortsFixture()
			Expect((*actual)[0].Name).To(Equal("--service-http-port--"))
		})
	})

	Context("Test NewProbeFixture", func() {
		It("should work", func() {
			actual := NewProbeFixture("container-name")
			Expect(actual.TimeoutSeconds).To(Equal(int32(5)))
		})
	})

	Context("Test NewPodFixture", func() {
		It("should work", func() {
			actual := NewPodFixture("service-name", "namespace", "conatiner-name", int32(80))
			Expect(actual.Name).To(Equal("service-name"))
		})
	})

	Context("Test NewServiceFixture", func() {
		It("should work", func() {
			actual := NewServiceFixture()
			Expect(actual.Name).To(Equal("--service-name--"))
		})
	})

	Context("Test NewEndpointsFixture", func() {
		It("should work", func() {
			actual := NewEndpointsFixture()
			Expect(actual.Name).To(Equal("--service-name--"))
		})
	})

	Context("Test NewIngressTestFixture", func() {
		It("should work", func() {
			actual := NewIngressTestFixture("namespace", "ingress-name")
			Expect(actual.Name).To(Equal("ingress-name"))
		})
	})

	Context("Test NewIngressTestFixtureBasic", func() {
		It("should work", func() {
			actual := NewIngressTestFixtureBasic("namespace", "ingress-name", true)
			Expect(actual.Name).To(Equal("ingress-name"))
		})
	})

	Context("Test NewPodTestFixture", func() {
		It("should work", func() {
			actual := NewPodTestFixture("namespace", "pod-name")
			Expect(actual.Name).To(Equal("pod-name"))
		})
	})

	Context("Test NewSecretTestFixture", func() {
		It("should work", func() {
			actual := NewSecretTestFixture()
			Expect(actual.Name).To(Equal("--the-name-of-the-secret--"))
		})
	})

})
