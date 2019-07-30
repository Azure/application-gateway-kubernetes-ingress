// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
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
})
