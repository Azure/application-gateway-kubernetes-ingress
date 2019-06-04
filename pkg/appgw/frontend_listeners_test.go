package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Process ingress rules and parse frontend listener configs", func() {

	listener80 := listenerIdentifier{
		FrontendPort: int32(80),
		HostName:     testFixturesHost,
	}

	listenerAzConfigNoSSL := listenerAzConfig{
		Protocol: "Http",
		Secret: secretIdentifier{
			Namespace: "",
			Name:      "",
		},
		SslRedirectConfigurationName: "",
	}

	Context("ingress rules without certificates", func() {
		certs := NewCertsFixture()
		cb := NewConfigBuilderFixture(&certs)
		ingress := NewIngressFixture()
		ingressList := []*v1beta1.Ingress{ingress}
		httpListenersAzureConfigMap := cb.getListenerConfigs(ingressList)

		It("should construct the App Gateway listeners correctly without SSL", func() {
			azConfigMapKeys := getMapKeys(&httpListenersAzureConfigMap)
			Expect(len(azConfigMapKeys)).To(Equal(2))
			Expect(azConfigMapKeys).To(ContainElement(listener80))
			actualVal := httpListenersAzureConfigMap[listener80]
			Expect(actualVal).To(Equal(listenerAzConfigNoSSL))
		})
	})
	Context("two ingresses with multiple ports", func() {
		certs := NewCertsFixture()
		cb := NewConfigBuilderFixture(&certs)

		ing1 := NewIngressFixture()
		ing2 := NewIngressFixture()
		ingressList := []*v1beta1.Ingress{
			ing1,
			ing2,
		}

		// !! Action !!
		listeners, _ := cb.getListeners(ingressList)

		It("should have correct number of listeners", func() {
			Expect(len(*listeners)).To(Equal(2))
		})

		It("should have correct values for listeners", func() {
			// Get the HTTPS listener for this test
			var listener n.ApplicationGatewayHTTPListener
			for _, listener = range *listeners {
				if listener.Protocol == "Https" && *listener.HostName == testFixturesHost {
					break
				}
			}

			Expect(*listener.HostName).To(Equal(testFixturesHost))
			Expect(*listener.FrontendPort.ID).To(Equal(cb.appGwIdentifier.frontendPortID(generateFrontendPortName(443))))

			expectedProtocol := n.ApplicationGatewayProtocol("Https")
			Expect(listener.Protocol).To(Equal(expectedProtocol))

			Expect(*listener.FrontendIPConfiguration.ID).To(Equal(testFixtureIPID1))
		})
	})
	Context("create a new App Gateway HTTP Listener", func() {
		It("should create a correct App Gwy listener", func() {
			certs := NewCertsFixture()
			cb := NewConfigBuilderFixture(&certs)
			listener := cb.newListener(listener80, n.ApplicationGatewayProtocol("Https"))
			expectedName := agPrefix + "fl-bye.com-80"

			expected := n.ApplicationGatewayHTTPListener{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(expectedName),
				ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
					// TODO: expose this to external configuration
					FrontendIPConfiguration: resourceRef(testFixtureIPID1),
					FrontendPort:            resourceRef(cb.appGwIdentifier.frontendPortID(generateFrontendPortName(80))),
					Protocol:                n.ApplicationGatewayProtocol("Https"),
					HostName:                to.StringPtr(testFixturesHost),
				},
			}

			Expect(listener).To(Equal(expected))
		})
	})
})
