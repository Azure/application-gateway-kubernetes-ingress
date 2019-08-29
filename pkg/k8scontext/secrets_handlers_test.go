// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	v1 "k8s.io/api/core/v1"
	"time"

	"github.com/onsi/ginkgo"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	istioFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
)

var _ = ginkgo.Describe("K8scontext Secrets Cache Handlers", func() {
	var k8sClient kubernetes.Interface

	ginkgo.Context("Test secrets handlers", func() {
		h := handlers{
			context: NewContext(k8sClient, fake.NewSimpleClientset(), istioFake.NewSimpleClientset(), []string{"ns"}, 1000*time.Second),
		}

		ginkgo.It("add, delete, update secrets from cache", func() {
			secret := &v1.Secret{}
			h.secretAdd(secret)
			h.secretDelete(secret)
			h.secretUpdate(secret, secret)
		})
	})
})
