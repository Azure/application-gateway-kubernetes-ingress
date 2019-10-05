// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

var _ = Describe("Test ConfigBuilderContext", func() {
	var cbCtx *ConfigBuilderContext
	ingress1 := tests.NewIngressTestFixture("ns1", "n1")
	ingress2 := tests.NewIngressTestFixture("ns2", "n2")
	ingress3 := tests.NewIngressTestFixture("ns3", "n3")
	Context("test InIngressList", func() {
		It("makes sure that ingress with different name or namespace is not true", func() {
			cbCtx = &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					&ingress1,
					&ingress2,
				},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			Expect(cbCtx.InIngressList(&ingress1)).To(BeTrue())
			Expect(cbCtx.InIngressList(&ingress2)).To(BeTrue())
			Expect(cbCtx.InIngressList(&ingress3)).To(BeFalse())

			cbCtx.IngressList = append(cbCtx.IngressList, &ingress3)

			Expect(cbCtx.InIngressList(&ingress3)).To(BeTrue())
		})
	})
})
