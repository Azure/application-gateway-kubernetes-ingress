// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"time"

	"github.com/onsi/ginkgo"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	istioFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = ginkgo.Describe("K8scontext Ingress Cache Handlers", func() {
	var k8sClient kubernetes.Interface

	ginkgo.Context("Test ingress handlers", func() {
		h := handlers{
			context: NewContext(k8sClient, fake.NewSimpleClientset(), istioFake.NewSimpleClientset(), []string{"ns"}, 1000*time.Second),
		}

		ginkgo.It("add, delete, update ingress from cache", func() {
			ing := fixtures.GetIngress()
			h.ingressAdd(ing)
			h.ingressDelete(ing)
			h.ingressUpdate(ing, ing)
		})
	})
})
