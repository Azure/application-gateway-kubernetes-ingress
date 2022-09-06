// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"context"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	multiClusterFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned/fake"
	istioFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/go-autorest/autorest/to"
)

var _ = ginkgo.Describe("K8scontext Ingress Cache Handlers", func() {
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

		secret := tests.NewSecretTestFixture()
		secret.Namespace = ""
		_, err = k8sClient.CoreV1().Secrets("ns").Create(context.TODO(), secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())
		_, err = k8sClient.CoreV1().Secrets("ns1").Create(context.TODO(), secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		class := tests.NewIngressClassFixture()
		_, err = k8sClient.NetworkingV1().IngressClasses().Create(context.TODO(), class, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		IsNetworkingV1PackageSupported = true
		ctx = NewContext(k8sClient, fake.NewSimpleClientset(), multiClusterFake.NewSimpleClientset(), istioFake.NewSimpleClientset(), []string{"ns"}, 1000*time.Second, metricstore.NewFakeMetricStore(), environment.GetFakeEnv())
		h = handlers{
			context: ctx,
		}
	})

	ginkgo.Context("Test ingress handlers", func() {
		ginkgo.It("add, delete, update ingress from cache for allowed namespace ns", func() {
			Expect(ctx.namespaces).ToNot(BeNil())
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
			Expect(ctx.namespaces).ToNot(BeNil())
			ing := fixtures.GetIngress()
			ing.Namespace = "ns1"
			h.ingressAdd(ing)
			Expect(len(h.context.Work)).To(Equal(0))
			h.ingressDelete(ing)
			Expect(len(h.context.Work)).To(Equal(0))
			h.ingressUpdate(ing, ing)
			Expect(len(h.context.Work)).To(Equal(0))
		})

		ginkgo.It("add, delete, update ingress from cache for allowed namespace ns", func() {
			Expect(ctx.namespaces).ToNot(BeNil())
			ing := fixtures.GetIngress()
			ing.Namespace = "ns"

			// use ingress class
			ing.Annotations[annotations.IngressClassKey] = ""
			ing.Spec.IngressClassName = to.StringPtr(environment.DefaultIngressClassResourceName)

			h.ingressAdd(ing)
			Expect(len(h.context.Work)).To(Equal(1))
			h.ingressDelete(ing)
			Expect(len(h.context.Work)).To(Equal(2))
			h.ingressUpdate(ing, ing)
			Expect(len(h.context.Work)).To(Equal(2))
		})

		ginkgo.It("should update the ingressSecretsMap even when secret is malformed", func() {
			namespace := "ns"
			data := map[string][]byte{
				"tls.crt": []byte(""),
				"tls.key": []byte(""),
			}
			secret := &v1.Secret{
				Type: "kubernetes.io/tls",
				ObjectMeta: metav1.ObjectMeta{
					Name:      tests.NameOfSecret,
					Namespace: namespace,
				},
				Data: data,
			}

			// create a malformed secret
			err := h.context.Caches.Secret.Add(secret)
			Expect(err).To(BeNil())

			secKey := utils.GetResourceKey(secret.Namespace, secret.Name)
			secretInterface, exists, err := h.context.Caches.Secret.GetByKey(secKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			cachedSecret := secretInterface.(*v1.Secret)
			Expect(cachedSecret.Data).To(Equal(data))

			ing := tests.NewIngressTestFixtureBasic(namespace, "ing", true)

			// add ingress
			h.ingressAdd(ing)

			// check that map is updated with the new key
			Expect(h.context.ingressSecretsMap.ContainsValue(secKey)).To(BeTrue())
		})
	})
})
