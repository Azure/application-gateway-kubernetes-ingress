// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

//go:build unittest
// +build unittest

package k8scontext

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	agiccrdFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	mcsfake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned/fake"
	istiofake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
)

var _ = ginkgo.Describe("Tests IngressBackend with nil Service (Resource backend)", func() {
	ginkgo.It("does not panic in IsEndpointReferencedByAnyIngress or IsPodReferencedByAnyIngress when an Ingress uses a Resource backend", func() {
		apiGroup := "example.com"
		resourceIngress := &networking.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   "tenant-a",
				Name:        "resource-backend-ingress",
				Annotations: map[string]string{annotations.IngressClassKey: environment.DefaultIngressClassController},
			},
			Spec: networking.IngressSpec{
				Rules: []networking.IngressRule{{
					Host: "resource.example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{{
								Path: "/",
								Backend: networking.IngressBackend{
									Resource: &v1.TypedLocalObjectReference{APIGroup: &apiGroup, Kind: "StorageBucket", Name: "path-bucket"},
								},
							}},
						},
					},
				}},
			},
		}

		k8sClient := testclient.NewSimpleClientset()
		ctx := context.Background()
		_, _ = k8sClient.CoreV1().Namespaces().Create(ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"}}, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Namespaces().Create(ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "prod"}}, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Services("prod").Create(ctx, &v1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "unrelated-svc"}}, metav1.CreateOptions{})
		_, _ = k8sClient.NetworkingV1().Ingresses("tenant-a").Create(ctx, resourceIngress, metav1.CreateOptions{})

		stopCh := make(chan struct{})
		defer close(stopCh)
		env := environment.GetFakeEnv()
		prevV1 := IsNetworkingV1PackageSupported
		IsNetworkingV1PackageSupported = true
		defer func() { IsNetworkingV1PackageSupported = prevV1 }()

		ctxt := NewContext(k8sClient, agiccrdFake.NewSimpleClientset(), mcsfake.NewSimpleClientset(), istiofake.NewSimpleClientset(), []string{}, 1000*time.Second, metricstore.NewFakeMetricStore(), env)
		Expect(ctxt.Run(stopCh, true, env)).To(Succeed())

		ep := &v1.Endpoints{ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "unrelated-svc"}}
		Expect(func() {
			Expect(ctxt.IsEndpointReferencedByAnyIngress(ep)).To(BeFalse())
		}).ToNot(Panic())
	})
})
