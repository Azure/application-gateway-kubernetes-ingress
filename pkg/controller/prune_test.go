// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("prune function tests", func() {
	var controller *AppGwIngressController

	BeforeEach(func() {
		controller = &AppGwIngressController{
			appGwIdentifier: appgw.Identifier{
				SubscriptionID: "xxxx",
				ResourceGroup:  "xxxx",
				AppGwName:      "appgw",
			},
			recorder: record.NewFakeRecorder(100),
		}
	})

	Context("ensure pruneNoPrivateIP prunes ingress", func() {
		ingressPrivate := tests.NewIngressFixture()
		ingressPrivate.Annotations = map[string]string{
			annotations.UsePrivateIPKey: "true",
		}
		ingressPublic := tests.NewIngressFixture()
		cbCtx := &appgw.ConfigBuilderContext{
			IngressList: []*v1beta1.Ingress{
				ingressPrivate,
				ingressPublic,
			},
			ServiceList: []*v1.Service{
				tests.NewServiceFixture(),
			},
		}
		appGw := fixtures.GetAppGateway()

		It("removes the ingress using private ip and keeps others", func() {
			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoPrivateIP(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(1))
			Expect(prunedIngresses[0].Annotations[annotations.UsePrivateIPKey]).To(Equal(""))
		})

		It("keeps the ingress using private ip when public ip is present", func() {
			appGw.FrontendIPConfigurations = &[]n.ApplicationGatewayFrontendIPConfiguration{
				fixtures.GetPublicIPConfiguration(),
				fixtures.GetPrivateIPConfiguration(),
			}
			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoPrivateIP(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(2))
		})
	})

	Context("ensure pruneRedirectNoTLS prunes ingress", func() {
		// invalid ingress without https and redirect
		ingressInvalid := tests.NewIngressFixture()
		ingressInvalid.Annotations = map[string]string{
			annotations.SslRedirectKey: "true",
		}
		ingressInvalid.Spec.TLS = nil
		It("should have ingressInvalid without https and redirect", func() {
			Expect(annotations.IsSslRedirect(ingressInvalid)).To(BeTrue())
			Expect(ingressInvalid.Spec.TLS).To(BeNil())
		})

		// valid ingress with https and redirect
		ingressValid1 := tests.NewIngressFixture()
		ingressValid1.Annotations = map[string]string{
			annotations.SslRedirectKey: "true",
		}
		It("should have ingressValid1 without https and redirect", func() {
			Expect(annotations.IsSslRedirect(ingressValid1)).To(BeTrue())
			Expect(ingressValid1.Spec.TLS).To(Not(BeNil()))
		})

		// valid ingress without https and redirect
		ingressValid2 := tests.NewIngressFixture()
		ingressValid2.Annotations = map[string]string{
			annotations.SslRedirectKey: "false",
		}
		ingressValid2.Spec.TLS = nil
		It("should have ingressValid2 without https and redirect", func() {
			Expect(annotations.IsSslRedirect(ingressValid2)).To(BeFalse())
			Expect(ingressValid2.Spec.TLS).To(BeNil())
		})

		cbCtx := &appgw.ConfigBuilderContext{
			IngressList: []*v1beta1.Ingress{
				ingressInvalid,
				ingressValid1,
				ingressValid2,
			},
			ServiceList: []*v1.Service{
				tests.NewServiceFixture(),
			},
		}
		appGw := fixtures.GetAppGateway()
		It("removes the invalid ingresses", func() {
			prunedIngresses := pruneRedirectWithNoTLS(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(cbCtx.IngressList)).To(Equal(3))
			Expect(len(prunedIngresses)).To(Equal(2))
			Expect(prunedIngresses).To(Not(ContainElement(ingressInvalid)))
			Expect(prunedIngresses).To(ContainElement(ingressValid1))
			Expect(prunedIngresses).To(ContainElement(ingressValid2))
		})
	})
})
