package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"k8s.io/client-go/tools/cache"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMakeHostToSecretMap(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up SSL redirect annotations")
}

var _ = Describe("makeHostToSecretMap", func() {
	Context("looking at TLS specs", func() {
		cb := makeConfigBuilderTestFixture(nil)

		// Cache a secret
		cacheKey := getResourceKey(testFixturesNamespace, testFixturesNameOfSecret)
		const cacheValue = "xyz"
		cc := cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{})
		cc.Add(cacheKey, []byte(cacheValue))
		cb.k8sContext.CertificateSecretStore = &k8scontext.SecretsStore{
			Cache: cc,
		}
		_ = cb.k8sContext.Caches.Secret.Add(cacheKey)

		ingress := makeIngressTestFixture()

		// !! Action !!
		actual := cb.makeHostToSecretMap(&ingress)

		expected := map[string]secretIdentifier{
			"ftp.contoso.com": {
				testFixturesNamespace,
				testFixturesNameOfSecret,
			},
			"www.contoso.com": {
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
		It("should succeed", func() {
			Expect(actual).To(Equal(expected))
		})
	})

	Context("running", func() {
		ingress := makeIngressTestFixture()
		cb := makeConfigBuilderTestFixture(nil)

		var secrets map[string]secretIdentifier

		// !! Action !!
		secrets = cb.makeHostToSecretMap(&ingress)

		It("should have exact number of keys", func() {
			Expect(len(secrets)).To(Equal(4))
		})

		It("should have correct keys", func() {
			var keys []string
			for k := range secrets {
				keys = append(keys, k)
			}
			Expect(keys).To(ContainElement(testFixturesHost))
			Expect(keys).To(ContainElement("www.contoso.com"))
			Expect(keys).To(ContainElement("ftp.contoso.com"))
			Expect(keys).To(ContainElement(""))
		})

		It("has the correct secrets", func() {
			expected := secretIdentifier{
				Namespace: testFixturesNamespace,
				Name:      testFixturesNameOfSecret,
			}
			Expect(secrets[testFixturesHost]).To(Equal(expected))
		})
	})

	Context("extract host to secret map", func() {
		certs := getCertsTestFixture()
		cb := makeConfigBuilderTestFixture(&certs)

		ing1 := makeIngressTestFixture()
		ing1.Annotations[annotations.SslRedirectKey] = "true"

		secrets := cb.makeHostToSecretMap(&ing1)

		It("ingress should be setup with 2 TLS records", func() {
			Expect(len(ing1.Spec.TLS)).To(Equal(2))
		})
		It("verify setup", func() {
			tls := ing1.Spec.TLS[0]
			tlsSecret := secretIdentifier{
				Name:      tls.SecretName,
				Namespace: ing1.Namespace,
			}
			expected := testFixturesNamespace + "/" + testFixturesNameOfSecret
			Expect(tlsSecret.secretKey()).To(Equal(expected))
			cert := cb.k8sContext.CertificateSecretStore.GetPfxCertificate(tlsSecret.secretKey())
			Expect(cert).To(Equal([]byte("xyz")))
		})
		It("the map should have 2 records", func() {
			Expect(len(secrets)).To(Equal(4))
		})
	})
})
