package appgw

import (
	"testing"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

func TestFrontendListeners(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up SSL redirect annotations")
}

var _ = Describe("Process ingress rules and parse front end listener config", func() {
	Context("with many frontend ports", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)

		ing1 := newIngressFixture()
		ing1.Annotations[annotations.SslRedirectKey] = "true"
		ing2 := newIngressFixture()
		ing2.Annotations[annotations.SslRedirectKey] = "true"
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
			// TODO(draychev): add a subscription to these tests
			testSubscription := ""

			// TODO(draychev): add a resource group to these tests
			testResourceGroup := ""

			// TODO(draychev): add an app gateway name to these tests
			testAppGateway := ""

			Expect(*listener.HostName).To(Equal(testFixturesHost))
			expectedPortID := "/subscriptions/" + testSubscription +
				"/resourceGroups/" + testResourceGroup +
				"/providers/Microsoft.Network/" +
				"applicationGateways/" + testAppGateway +
				"/frontEndPorts/k8s-ag-ingress-fp-443"
			Expect(*listener.FrontendPort.ID).To(Equal(expectedPortID))
		})
	})
})
