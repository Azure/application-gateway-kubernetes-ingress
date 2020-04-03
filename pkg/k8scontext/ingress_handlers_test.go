// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	istioFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = ginkgo.Describe("K8scontext Ingress Cache Handlers", func() {
	var k8sClient kubernetes.Interface
	var context *Context
	var h handlers

	ginkgo.BeforeEach(func() {
		k8sClient = testclient.NewSimpleClientset()

		_, err := k8sClient.CoreV1().Namespaces().Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = k8sClient.CoreV1().Namespaces().Create(&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns1",
			},
		})
		Expect(err).ToNot(HaveOccurred())

		secret := tests.NewSecretTestFixture()
		secret.Namespace = ""
		_, err = k8sClient.CoreV1().Secrets("ns").Create(secret)
		Expect(err).To(BeNil())
		_, err = k8sClient.CoreV1().Secrets("ns1").Create(secret)
		Expect(err).To(BeNil())

		IsNetworkingV1Beta1PackageSupported = true
		context = NewContext(k8sClient, fake.NewSimpleClientset(), istioFake.NewSimpleClientset(), []string{"ns"}, 1000*time.Second, metricstore.NewFakeMetricStore())
		h = handlers{
			context: context,
		}
	})

	ginkgo.Context("Test ingress handlers", func() {
		ginkgo.It("add, delete, update ingress from cache for allowed namespace ns", func() {
			Expect(context.namespaces).ToNot(BeNil())
			ing := fixtures.GetIngress()
			ing.Namespace = "ns"
			h.ingressAdd(ing)
			Expect(len(h.context.Work)).To(Equal(1))
			h.ingressDelete(ing)
			Expect(len(h.context.Work)).To(Equal(2))
			h.ingressUpdate(ing, ing)
			Expect(len(h.context.Work)).To(Equal(2))
		})

		ginkgo.It("should not add events for namespace ns1 not in the namespaces list", func() {
			Expect(context.namespaces).ToNot(BeNil())
			ing := fixtures.GetIngress()
			ing.Namespace = "ns1"
			h.ingressAdd(ing)
			Expect(len(h.context.Work)).To(Equal(0))
			h.ingressDelete(ing)
			Expect(len(h.context.Work)).To(Equal(0))
			h.ingressUpdate(ing, ing)
			Expect(len(h.context.Work)).To(Equal(0))
		})
	})
})
