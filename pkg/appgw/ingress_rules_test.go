package appgw

import (
	"testing"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

func TestIngressRules(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test parsing of ingress rules")
}

var _ = Describe("Process ingress rules, listeners, and ports", func() {
	port80 := int32(80)
	port443 := int32(443)

	expectedListener80 := frontendListenerIdentifier{
		FrontendPort: port80,
		HostName:     testFixturesHost,
	}

	expectedListenerAzConfigNoSSL := frontendListenerAzureConfig{
		Protocol: "Http",
		Secret: secretIdentifier{
			Namespace: "",
			Name:      "",
		},
		SslRedirectConfigurationName: "",
	}

	expectedListener443 := frontendListenerIdentifier{
		FrontendPort: 443,
		HostName:     testFixturesHost,
	}

	expectedListenerAzConfigSSL := frontendListenerAzureConfig{
		Protocol: "Https",
		Secret: secretIdentifier{
			Namespace: testFixturesNamespace,
			Name:      testFixturesNameOfSecret,
		},
		SslRedirectConfigurationName: agPrefix + "-" +
			testFixturesNamespace +
			"-" +
			testFixturesName +
			"-sslr",
	}

	Context("ingress rules without certificates", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)
		ingress := newIngressFixture()
		ingressList := []*v1beta1.Ingress{ingress}
		httpListenersAzureConfigMap := cb.getListenerConfigs(ingressList)

		// Ensure there are no certs
		ingress.Spec.TLS = nil

		// !! Action !!
		frontendListeners, frontendPorts, _ := cb.processIngressRules(ingress)

		// Verify front end listeners
		It("should have correct count of frontend listeners", func() {
			Expect(len(frontendPorts.ToSlice())).To(Equal(1))
		})
		It("should have a listener on port 80", func() {
			actualListener := (frontendListeners.ToSlice()[0]).(frontendListenerIdentifier)
			Expect(actualListener).To(Equal(expectedListener80))
		})

		// Verify front end ports
		It("should have correct count of front end ports", func() {
			Expect(len(frontendPorts.ToSlice())).To(Equal(1))
		})

		It("should have one port 80", func() {
			actualPort := frontendPorts.ToSlice()[0]
			Expect(actualPort).To(Equal(port80))
		})

		// check the request routing rules
		It("should have no request routing rules", func() {
			Expect(cb.appGwConfig.RequestRoutingRules).To(BeNil())
		})

		It("should construct the App Gateway listeners correctly without SSL", func() {
			azConfigMapKeys := getMapKeys(&httpListenersAzureConfigMap)
			Expect(len(azConfigMapKeys)).To(Equal(2))
			Expect(azConfigMapKeys).To(ContainElement(expectedListener80))

			actualVal := httpListenersAzureConfigMap[expectedListener80]
			Expect(*actualVal).To(Equal(expectedListenerAzConfigNoSSL))
		})
	})

	Context("ingress rules with certificates", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)
		ingress := newIngressFixture()
		ingressList := []*v1beta1.Ingress{ingress}
		It("should have setup tests with some TLS certs", func() {
			Ω(len(ingress.Spec.TLS)).Should(BeNumerically(">=", 2))
		})

		// !! Action !!
		httpListenersAzureConfigMap := cb.getListenerConfigs(ingressList)

		It("should configure App Gateway listeners correctly with SSL", func() {
			azConfigMapKeys := getMapKeys(&httpListenersAzureConfigMap)
			Expect(len(azConfigMapKeys)).To(Equal(2))
			Expect(azConfigMapKeys).To(ContainElement(expectedListener443))

			actualVal := httpListenersAzureConfigMap[expectedListener443]
			Expect(*actualVal).To(Equal(expectedListenerAzConfigSSL))
		})
	})

	Context("with attached certificates", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)
		ingress := newIngressFixture()
		ingress.Annotations[annotations.SslRedirectKey] = "one/two/three"

		// !! Action !!
		frontendListeners, frontendPorts, _ := cb.processIngressRules(ingress)

		ingressList := []*v1beta1.Ingress{ingress}
		httpListenersAzureConfigMap := cb.getListenerConfigs(ingressList)

		It("should have correct number of front end listener", func() {
			Expect(len(frontendListeners.ToSlice())).To(Equal(1))
		})
		It("should have correct number of front end ports", func() {
			Expect(len(frontendPorts.ToSlice())).To(Equal(1))
		})
		It("should have a listener on port 443", func() {
			actualListener := (frontendListeners.ToSlice()[0]).(frontendListenerIdentifier)
			Expect(actualListener.FrontendPort).To(Equal(port443))
		})
		It("should have one port 443", func() {
			actualPort := frontendPorts.ToSlice()[0]
			Expect(actualPort).To(Equal(port443))
		})

		It("should have no request routing rules ", func() {
			Expect(cb.appGwConfig.RequestRoutingRules).To(BeNil())
		})

		It("should configure App Gateway listeners correctly", func() {
			azConfigMapKeys := getMapKeys(&httpListenersAzureConfigMap)
			Expect(len(azConfigMapKeys)).To(Equal(1))
			Expect(azConfigMapKeys[0].FrontendPort).To(Equal(port443))

			actualVal := httpListenersAzureConfigMap[azConfigMapKeys[0]]
			Expect(*actualVal).To(Equal(expectedListenerAzConfigSSL))
		})
	})
})

func getMapKeys(m *map[frontendListenerIdentifier]*frontendListenerAzureConfig) []frontendListenerIdentifier {
	keys := make([]frontendListenerIdentifier, 0, len(*m))
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}
