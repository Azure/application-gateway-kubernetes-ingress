// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("test pruning Ingress based on white/white lists", func() {

	Context("Test PruneIngressRules()", func() {
		prohibited := fixtures.GetProhibitedTargets()

		ingress := v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						// Rule with no Paths
						Host: tests.OtherHost,
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{},
						},
					},
					{
						// Rule with Paths
						Host: tests.Host,
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: fixtures.PathFoo,
										Backend: v1beta1.IngressBackend{
											ServiceName: tests.ServiceName,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 80,
											},
										},
									},
									{
										Path: fixtures.PathFox,
										Backend: v1beta1.IngressBackend{
											ServiceName: tests.ServiceName,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 443,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		actualRules := PruneIngressRules(&ingress, prohibited)

		expected := v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						// Should have kept the rule with no Paths
						Host: tests.OtherHost,
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{},
						},
					},
					{
						// Should have kept one of the Paths of this Rule
						Host: tests.Host,
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: fixtures.PathFoo,
										Backend: v1beta1.IngressBackend{
											ServiceName: tests.ServiceName,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		It("should have trimmed the ingress rules to what AGIC is allowed to manage", func() {
			Expect(actualRules).To(Equal(expected.Spec.Rules))
		})
	})

	Context("Test canManage()", func() {
		blacklist := []Target{{
			Hostname: tests.Host,
			Path:     to.StringPtr(fixtures.PathFox),
		}}

		It("should have properly identified the ingress rules AGIC is NOT allowed to manage", func() {
			actual := canManage(tests.Host, to.StringPtr(fixtures.PathFox), &blacklist)
			Expect(actual).To(BeFalse())
		})

		It("should have properly identified the ingress rules AGIC is allowed to manage", func() {
			actual := canManage(tests.Host, to.StringPtr(fixtures.PathBaz), &blacklist)
			Expect(actual).To(BeTrue())
		})
	})

})
