package appgw

import (
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("MutateAppGateway ingress rules, listeners, and ports", func() {
	port443 := Port(443)

	expectedListener80, _ := newTestListenerID(Port(80), []string{tests.Host}, false)

	expectedListenerAzConfigNoSSL := listenerAzConfig{
		Protocol: "Http",
		Secret: secretIdentifier{
			Namespace: "",
			Name:      "",
		},
		SslRedirectConfigurationName: "",
	}

	expectedListener443, expectedListener443Name := newTestListenerID(Port(443), []string{tests.Host}, false)

	expectedListenerAzConfigSSL := listenerAzConfig{
		Protocol: "Https",
		Secret: secretIdentifier{
			Namespace: tests.Namespace,
			Name:      tests.NameOfSecret,
		},
		SslRedirectConfigurationName: "sslr-" + expectedListener443Name,
	}

	Context("ingress rules without certificates", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)
		ingress := tests.NewIngressFixture()
		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}
		listenersAzureConfigMap := cb.getListenerConfigs(cbCtx)

		// Ensure there are no certs
		ingress.Spec.TLS = nil

		// !! Action !!
		listenerConfigs := cb.getListenersFromIngress(ingress, cbCtx.EnvVariables, nil)

		// Verify front end listeners
		It("should have correct count of frontend listeners", func() {
			Expect(len(listenerConfigs)).To(Equal(1))
		})
		It("should have a listener on port 80", func() {
			actualListenerID := getMapKeys(&listenerConfigs)[0]
			Expect(actualListenerID).To(Equal(expectedListener80))
		})

		// check the request routing rules
		It("should have no request routing rules", func() {
			Expect(cb.appGw.RequestRoutingRules).To(BeNil())
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
		ingress := tests.NewIngressFixture()
		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}
		It("should have setup tests with some TLS certs", func() {
			Ω(len(ingress.Spec.TLS)).Should(BeNumerically(">=", 2))
		})

		// !! Action !!
		httpListenersAzureConfigMap := cb.getListenerConfigs(cbCtx)

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
		ingress := tests.NewIngressFixture()
		ingress.Annotations[annotations.SslRedirectKey] = "one/two/three"
		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		// !! Action !!
		frontendListeners := cb.getListenersFromIngress(ingress, cbCtx.EnvVariables, nil)

		httpListenersAzureConfigMap := cb.getListenerConfigs(cbCtx)

		It("should have correct number of front end listener", func() {
			Expect(len(frontendListeners)).To(Equal(1))
		})
		It("should have a listener on port 443", func() {
			listeners := getMapKeys(&frontendListeners)
			ports := make([]Port, 0, len(listeners))
			for _, listener := range listeners {
				ports = append(ports, listener.FrontendPort)
			}
			Expect(ports).To(ContainElement(port443))
		})

		It("should have no request routing rules ", func() {
			Expect(cb.appGw.RequestRoutingRules).To(BeNil())
		})

		It("should configure App Gateway listeners correctly", func() {
			azConfigMapKeys := getMapKeys(&httpListenersAzureConfigMap)
			Expect(len(azConfigMapKeys)).To(Equal(1))
			Expect(azConfigMapKeys[0].FrontendPort).To(Equal(port443))

			actualVal := httpListenersAzureConfigMap[azConfigMapKeys[0]]
			expectedListenerConfig := listenerAzConfig{
				Protocol: "Https",
				Secret: secretIdentifier{
					Namespace: tests.Namespace,
					Name:      tests.NameOfSecret,
				},
			}
			Expect(actualVal).To(Equal(expectedListenerConfig))
			Expect(actualVal.SslRedirectConfigurationName).To(Equal(""))
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

func getPortsList(m *map[Port]interface{}) []Port {
	ports := make([]Port, 0, len(*m))
	for port := range *m {
		ports = append(ports, port)
	}
	return ports
}
