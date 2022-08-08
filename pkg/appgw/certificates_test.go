package appgw

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Testing function newHostToSecretMap", func() {
	const host1 = "ftp.contoso.com"
	const host2 = "www.contoso.com"
	expectedHostToSecretMap := map[string]secretIdentifier{
		host1: {
			tests.Namespace,
			tests.NameOfSecret,
		},
		host2: {
			tests.Namespace,
			tests.NameOfSecret,
		},
		tests.Host: {
			tests.Namespace,
			tests.NameOfSecret,
		},
		"": {
			tests.Namespace,
			tests.NameOfSecret,
		},
	}

	expectedSecret := secretIdentifier{
		Namespace: tests.Namespace,
		Name:      tests.NameOfSecret,
	}

	Context("Test fetching secrets from ingress with TLS spec", func() {
		cb := newConfigBuilderFixture(nil)
		ingress := tests.NewIngressFixture()

		actualHostToSecretMap := cb.newHostToSecretMap(ingress)

		It("should have generated the expected host to secret map", func() {
			Expect(actualHostToSecretMap).To(Equal(expectedHostToSecretMap))
		})
		It("should have correct keys", func() {
			var keys []string
			for k := range actualHostToSecretMap {
				keys = append(keys, k)
			}

			// We check each key to ensure that unstable sort does not cause test flakiness
			Expect(keys).To(ContainElement(tests.Host))
			Expect(keys).To(ContainElement(host1))
			Expect(keys).To(ContainElement(host2))
			Expect(keys).To(ContainElement(""))
		})

		It("has the correct secrets", func() {
			Expect(actualHostToSecretMap[tests.Host]).To(Equal(expectedSecret))
		})
	})

	Context("Test obtaining a single certificate for an existing host", func() {
		cb := newConfigBuilderFixture(nil)
		ingress := tests.NewIngressFixture()
		hostnameSecretIDMap := cb.newHostToSecretMap(ingress)
		actualSecret, actualSecretID := cb.getCertificate(ingress, host1, hostnameSecretIDMap)

		It("should have generated the expected secret", func() {
			Expect(*actualSecret).To(Equal("eHl6"))
		})

		It("should have generated the correct secretID struct", func() {
			Expect(*actualSecretID).To(Equal(expectedSecret))
		})
	})
})
