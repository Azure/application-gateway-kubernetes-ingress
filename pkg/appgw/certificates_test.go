package appgw

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNewHostToSecretMap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test newHostToSecretMap function")
}

var _ = Describe("Testing function newHostToSecretMap", func() {
	const host1 = "ftp.contoso.com"
	const host2 = "www.contoso.com"
	expectedHostToSecretMap := map[string]secretIdentifier{
		host1: {
			testFixturesNamespace,
			testFixturesNameOfSecret,
		},
		host2: {
			testFixturesNamespace,
			testFixturesNameOfSecret,
		},
		testFixturesHost: {
			testFixturesNamespace,
			testFixturesNameOfSecret,
		},
		"": {
			testFixturesNamespace,
			testFixturesNameOfSecret,
		},
	}

	expectedSecret := secretIdentifier{
		Namespace: testFixturesNamespace,
		Name:      testFixturesNameOfSecret,
	}

	Context("Test fetching secrets from ingress with TLS spec", func() {
		cb := makeConfigBuilderTestFixture(nil)
		ingress := makeIngressFixture()

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
			Expect(keys).To(ContainElement(testFixturesHost))
			Expect(keys).To(ContainElement(host1))
			Expect(keys).To(ContainElement(host2))
			Expect(keys).To(ContainElement(""))
		})

		It("has the correct secrets", func() {
			Expect(actualHostToSecretMap[testFixturesHost]).To(Equal(expectedSecret))
		})
	})

	Context("Test obtaining a single certificate for an existing host", func() {
		cb := makeConfigBuilderTestFixture(nil)
		ingress := makeIngressFixture()
		hostnameSecretIDMap := cb.newHostToSecretMap(ingress)
		actualSecret, actualSecretID := cb.getCertificateV1(ingress, host1, hostnameSecretIDMap)

		It("should have generated the expected secret", func() {
			Expect(*actualSecret).To(Equal("eHl6"))
		})

		It("should have generated the correct secretID struct", func() {
			Expect(*actualSecretID).To(Equal(expectedSecret))
		})
	})
})
