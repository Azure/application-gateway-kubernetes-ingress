package appgw

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

func TestFrontendListeners(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up SSL redirect annotations")
}

var _ = Describe("Process ingress rules and parse frontend listener configs", func() {
	Context("ingress rules without certificates", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)
		ingress := newIngressFixture()
		ingressList := []*v1beta1.Ingress{ingress}
		httpListenersAzureConfigMap := cb.getListenerConfigs(ingressList)

		expectedListener80 := frontendListenerIdentifier{
			FrontendPort: int32(80),
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

		It("should construct the App Gateway listeners correctly without SSL", func() {
			azConfigMapKeys := getMapKeys(&httpListenersAzureConfigMap)
			Expect(len(azConfigMapKeys)).To(Equal(2))
			Expect(azConfigMapKeys).To(ContainElement(expectedListener80))
			actualVal := httpListenersAzureConfigMap[expectedListener80]
			Expect(*actualVal).To(Equal(expectedListenerAzConfigNoSSL))
		})
	})
	Context("two ingresses with multiple ports", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)

		ing1 := newIngressFixture()
		ing2 := newIngressFixture()
		ingressList := []*v1beta1.Ingress{
			ing1,
			ing2,
		}

		// !! Action !!
		listeners, _ := cb.getFrontendListeners(ingressList)

		It("should have correct number of listeners", func() {
			Expect(len(*listeners)).To(Equal(2))
		})

		It("should have correct values for listeners", func() {
			// Get the HTTPS listener for this test
			var listener network.ApplicationGatewayHTTPListener
			for _, listener = range *listeners {
				if listener.Protocol == "Https" && *listener.HostName == testFixturesHost {
					break
				}
			}

			Expect(*listener.HostName).To(Equal(testFixturesHost))
			fePortID := "k8s-ag-ingress-fp-443"
			expectedPortID := "/subscriptions/" + testFixtureSubscription +
				"/resourceGroups/" + testFixtureResourceGroup +
				"/providers/Microsoft.Network" +
				"/applicationGateways/" + testFixtureAppGwName +
				"/frontEndPorts/" + fePortID
			Expect(*listener.FrontendPort.ID).To(Equal(expectedPortID))

			expectedProtocol := network.ApplicationGatewayProtocol("Https")
			Expect(listener.Protocol).To(Equal(expectedProtocol))

			Expect(*listener.FrontendIPConfiguration.ID).To(Equal(testFixtureIPID1))
		})
	})
})
