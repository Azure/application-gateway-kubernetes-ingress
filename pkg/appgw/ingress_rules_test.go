package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Process ingress rules, listeners, and ports", func() {
	port80 := int32(80)
	port443 := int32(443)

	expectedListener80 := listenerIdentifier{
		FrontendPort: port80,
		HostName:     testFixturesHost,
	}

	expectedListenerAzConfigNoSSL := listenerAzConfig{
		Protocol: "Http",
		Secret: secretIdentifier{
			Namespace: "",
			Name:      "",
		},
		SslRedirectConfigurationName: "",
	}

	expectedListener443 := listenerIdentifier{
		FrontendPort: 443,
		HostName:     testFixturesHost,
	}

	expectedListenerAzConfigSSL := listenerAzConfig{
		Protocol: "Https",
		Secret: secretIdentifier{
			Namespace: testFixturesNamespace,
			Name:      testFixturesNameOfSecret,
		},
		SslRedirectConfigurationName: agPrefix +
			"sslr-" +
			testFixturesNamespace +
			"-" +
			testFixturesName,
	}

	Context("ingress rules without certificates", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)
		ingress := newIngressFixture()
		ingressList := []*v1beta1.Ingress{ingress}
		listenersAzureConfigMap := cb.getListenerConfigs(ingressList)

		// Ensure there are no certs
		ingress.Spec.TLS = nil

		// !! Action !!
		frontendPorts, listenerConfigs := cb.processIngressRules(ingress)

		// Verify front end listeners
		It("should have correct count of frontend listeners", func() {
			Expect(len(listenerConfigs)).To(Equal(1))
		})
		It("should have a listener on port 80", func() {
			actualListenerID := getMapKeys(&listenerConfigs)[0]
			Expect(actualListenerID).To(Equal(expectedListener80))
		})

		// Verify front end ports
		It("should have correct count of front end ports", func() {
			Expect(len(frontendPorts)).To(Equal(1))
		})

		It("should have one port 80", func() {
			actualPort := getInt32MapKeys(&frontendPorts)[0]
			Expect(actualPort).To(Equal(port80))
		})

		// check the request routing rules
		It("should have no request routing rules", func() {
			Expect(cb.appGwConfig.RequestRoutingRules).To(BeNil())
		})

		It("should construct the App Gateway listeners correctly without SSL", func() {
			azConfigMapKeys := getMapKeys(&listenersAzureConfigMap)
			Expect(len(azConfigMapKeys)).To(Equal(2))
			Expect(azConfigMapKeys).To(ContainElement(expectedListener80))
			actualVal := listenersAzureConfigMap[expectedListener80]
			Expect(actualVal).To(Equal(expectedListenerAzConfigNoSSL))
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
			Expect(actualVal).To(Equal(expectedListenerAzConfigSSL))
		})
	})

	Context("with attached certificates", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)
		ingress := newIngressFixture()
		ingress.Annotations[annotations.SslRedirectKey] = "one/two/three"

		// !! Action !!
		frontendPorts, frontendListeners := cb.processIngressRules(ingress)

		ingressList := []*v1beta1.Ingress{ingress}
		httpListenersAzureConfigMap := cb.getListenerConfigs(ingressList)

		It("should have correct number of front end listener", func() {
			Expect(len(frontendListeners)).To(Equal(1))
		})
		It("should have correct number of front end ports", func() {
			Expect(len(frontendPorts)).To(Equal(1))
		})
		It("should have a listener on port 443", func() {
			actualListener := getMapKeys(&frontendListeners)[0]
			Expect(actualListener.FrontendPort).To(Equal(port443))
		})
		It("should have one port 443", func() {
			actualPort := getInt32MapKeys(&frontendPorts)[0]
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
			Expect(actualVal).To(Equal(expectedListenerAzConfigSSL))
		})
	})
})

func getMapKeys(m *map[listenerIdentifier]listenerAzConfig) []listenerIdentifier {
	keys := make([]listenerIdentifier, 0, len(*m))
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}

func getInt32MapKeys(m *map[int32]interface{}) []int32 {
	keys := make([]int32, 0, len(*m))
	for k := range *m {
		keys = append(keys, k)
	}
	return keys
}
