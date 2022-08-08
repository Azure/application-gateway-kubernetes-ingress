// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	multiClusterFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned/fake"
	istioFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
)

var _ = ginkgo.Describe("K8scontext General Cache Handlers", func() {
	var k8sClient kubernetes.Interface
	var ctx *Context
	var h handlers

	ginkgo.BeforeEach(func() {
		k8sClient = testclient.NewSimpleClientset()

		_, err := k8sClient.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		_, err = k8sClient.CoreV1().Namespaces().Create(context.TODO(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns1",
			},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		IsNetworkingV1PackageSupported = true
		ctx = NewContext(k8sClient, fake.NewSimpleClientset(), multiClusterFake.NewSimpleClientset(), istioFake.NewSimpleClientset(), []string{"ns"}, 1000*time.Second, metricstore.NewFakeMetricStore(), environment.GetFakeEnv())
		h = handlers{
			context: ctx,
		}
	})

	ginkgo.Context("Test general handlers", func() {
		ginkgo.It("add, delete, update pods from cache for allowed namespace ns", func() {
			pod := tests.NewPodTestFixture("ns", "pod")
			ctx.ingressSecretsMap.Insert("ns/ingress", utils.GetResourceKey(pod.Namespace, pod.Name))

			h.addFunc(&pod)
			Expect(len(h.context.Work)).To(Equal(1))
			h.deleteFunc(&pod)
			Expect(len(h.context.Work)).To(Equal(2))
			h.updateFunc(&pod, &pod)
			Expect(len(h.context.Work)).To(Equal(2))
		})

		ginkgo.It("should not add pods for namespace ns1 not in the namespaces list", func() {
			pod := tests.NewPodTestFixture("ns1", "pod")
			ctx.ingressSecretsMap.Insert("ns1/ingress", utils.GetResourceKey(pod.Namespace, pod.Name))

			h.addFunc(&pod)
			Expect(len(h.context.Work)).To(Equal(0))
			h.deleteFunc(&pod)
			Expect(len(h.context.Work)).To(Equal(0))
			h.updateFunc(&pod, &pod)
			Expect(len(h.context.Work)).To(Equal(0))
		})
	})
})
