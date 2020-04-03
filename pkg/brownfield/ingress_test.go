// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	networking "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("test pruning Ingress based on white/white lists", func() {

	Context("Test PruneIngressRules()", func() {
		prohibited := fixtures.GetAzureIngressProhibitedTargets()

		ingress := networking.Ingress{
			Spec: networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						// Rule with no Paths
						Host: tests.OtherHost,
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{},
						},
					},
					{
						// Rule with Paths
						Host: tests.Host,
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: fixtures.PathFoo,
										Backend: networking.IngressBackend{
											ServiceName: tests.ServiceName,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 80,
											},
										},
									},
									{
										Path: fixtures.PathFox,
										Backend: networking.IngressBackend{
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

		expected := networking.Ingress{
			Spec: networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						// Should have kept one of the Paths of this Rule
						Host: tests.Host,
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: fixtures.PathFoo,
										Backend: networking.IngressBackend{
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

})
