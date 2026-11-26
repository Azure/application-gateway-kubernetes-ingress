// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

//go:build unittest
// +build unittest

package appgw

import (
	"context"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	multiCluster_fake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned/fake"
	istio_fake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/mocks"
)

var _ = Describe("Tests IngressBackend with nil Service (Resource backend)", func() {
	It("does not panic in PreBuildValidate or Build when an Ingress uses Resource backends and preserves valid Service Ingresses", func() {
		apiGroup := "example.com"
		resourceIngress := &networking.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   "tenant-a",
				Name:        "resource-backend-ingress",
				Annotations: map[string]string{annotations.IngressClassKey: environment.DefaultIngressClassController},
			},
			Spec: networking.IngressSpec{
				DefaultBackend: &networking.IngressBackend{
					Resource: &v1.TypedLocalObjectReference{APIGroup: &apiGroup, Kind: "StorageBucket", Name: "default-bucket"},
				},
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

		validIngress := tests.NewIngressTestFixture("default", "valid-ingress")

		k8sClient := testclient.NewSimpleClientset()
		ctx := context.Background()
		_, _ = k8sClient.CoreV1().Namespaces().Create(ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "tenant-a"}}, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Namespaces().Create(ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}, metav1.CreateOptions{})
		svc := tests.NewServiceFixture()
		_, _ = k8sClient.CoreV1().Services("default").Create(ctx, svc, metav1.CreateOptions{})
		_, _ = k8sClient.NetworkingV1().Ingresses("tenant-a").Create(ctx, resourceIngress, metav1.CreateOptions{})
		_, _ = k8sClient.NetworkingV1().Ingresses("default").Create(ctx, &validIngress, metav1.CreateOptions{})

		stopCh := make(chan struct{})
		defer close(stopCh)
		env := environment.GetFakeEnv()
		prevV1 := k8scontext.IsNetworkingV1PackageSupported
		k8scontext.IsNetworkingV1PackageSupported = true
		defer func() { k8scontext.IsNetworkingV1PackageSupported = prevV1 }()

		ctxt := k8scontext.NewContext(k8sClient, fake.NewSimpleClientset(), multiCluster_fake.NewSimpleClientset(), istio_fake.NewSimpleClientset(), []string{}, 1000*time.Second, metricstore.NewFakeMetricStore(), env)
		Expect(ctxt.Run(stopCh, true, env)).To(Succeed())

		appGwy := &n.ApplicationGateway{ApplicationGatewayPropertiesFormat: NewAppGwyConfigFixture()}
		cb := NewConfigBuilder(ctxt, &Identifier{SubscriptionID: tests.Subscription, ResourceGroup: tests.ResourceGroup, AppGwName: tests.AppGwName}, appGwy, record.NewFakeRecorder(100), mocks.Clock{})
		cbCtx := &ConfigBuilderContext{
			IngressList:           ctxt.ListHTTPIngresses(),
			ServiceList:           ctxt.ListServices(),
			EnvVariables:          env,
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		Expect(cb.PreBuildValidate(cbCtx)).To(Succeed())
		built, err := cb.Build(cbCtx)
		Expect(err).ToNot(HaveOccurred())
		Expect(built).ToNot(BeNil())
	})
})
