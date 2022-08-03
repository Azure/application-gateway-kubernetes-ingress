package convert

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

var _ = Describe("Test conversion functions", func() {
	ingressV1, _ := tests.GetIngressV1FromFile("testdata/ingress-v1.yaml")
	ingressV1Beta1, _ := tests.GetIngressV1Beta1FromFile("testdata/ingress-v1beta1.yaml")

	// remove PathType property as that will not be converted
	for ruleIdx := range ingressV1.Spec.Rules {
		for pathIdx := range ingressV1.Spec.Rules[ruleIdx].HTTP.Paths {
			ingressV1.Spec.Rules[ruleIdx].HTTP.Paths[pathIdx].PathType = nil
		}
	}

	Context("Test ingress converstions", func() {

		It("should have v1.Ingress with all properties set", func() {
			Expect(ingressV1.Spec.DefaultBackend).NotTo(BeNil())
			Expect(ingressV1.Spec.TLS).NotTo(BeNil())
			Expect(ingressV1.Spec.Rules).NotTo(BeNil())
			Expect(ingressV1.Status).NotTo(BeNil())
		})

		It("correctly converts v1beta.Ingress to v1.Ingress", func() {
			convertIngressV1, converted := ToIngressV1(ingressV1Beta1)
			Expect(converted, BeTrue())

			Expect(convertIngressV1).To(Equal(ingressV1))
		})

		It("should not match ingress v1.Ingress if properties are different", func() {
			convertIngressV1, converted := ToIngressV1(ingressV1Beta1)
			Expect(converted, BeTrue())

			convertIngressV1.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name = "random"
			Expect(convertIngressV1).ToNot(Equal(ingressV1))
		})
	})
})
