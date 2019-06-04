package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Process ingress rules", func() {
	Context("with many frontend ports", func() {
		certs := NewCertsFixture()
		cb := newConfigBuilderFixture(&certs)

		ingressList := []*v1beta1.Ingress{
			newIngressFixture(),
			newIngressFixture(),
		}

		ports := cb.getFrontendPorts(ingressList)

		It("should have correct count of ports", func() {
			Expect(len(*ports)).To(Equal(2))
		})

		It("should have port 80", func() {
			expected := network.ApplicationGatewayFrontendPort{
				ApplicationGatewayFrontendPortPropertiesFormat: &network.ApplicationGatewayFrontendPortPropertiesFormat{
					Port:              to.Int32Ptr(80),
					ProvisioningState: nil,
				},
				Name: to.StringPtr("fp-80"),
				Etag: to.StringPtr("*"),
				Type: nil,
				ID:   nil,
			}
			Expect(*ports).To(ContainElement(expected))
		})

		It("should have port 443", func() {
			expected := network.ApplicationGatewayFrontendPort{
				ApplicationGatewayFrontendPortPropertiesFormat: &network.ApplicationGatewayFrontendPortPropertiesFormat{
					Port:              to.Int32Ptr(443),
					ProvisioningState: nil,
				},
				Name: to.StringPtr("fp-443"),
				Etag: to.StringPtr("*"),
				Type: nil,
				ID:   nil,
			}
			Expect(*ports).To(ContainElement(expected))
		})
	})
})
