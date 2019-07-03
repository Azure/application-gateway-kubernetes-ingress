package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Process ingress rules", func() {
	Context("with many frontend ports", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)

		ingressList := []*v1beta1.Ingress{
			tests.NewIngressFixture(),
			tests.NewIngressFixture(),
		}

		cbCtx := ConfigBuilderContext{
			IngressList: ingressList,
		}

		ports := cb.getFrontendPorts(&cbCtx)

		It("should have correct count of ports", func() {
			Expect(len(*ports)).To(Equal(2))
		})

		It("should have port 80", func() {
			expected := n.ApplicationGatewayFrontendPort{
				ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
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
			expected := n.ApplicationGatewayFrontendPort{
				ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
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
