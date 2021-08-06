// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"context"
	"time"

	"k8s.io/client-go/tools/record"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	multiCluster_fake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned/fake"
	istio_fake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("process function tests", func() {
	var controller *AppGwIngressController
	var cbCtx *appgw.ConfigBuilderContext
	var appGw n.ApplicationGateway
	var k8sClient kubernetes.Interface
	var ctxt *k8scontext.Context
	var stopChannel chan struct{}
	var ingress *networking.Ingress
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: tests.Namespace,
		},
	}
	publicIP := k8scontext.IPAddress("xxxx")
	privateIP := k8scontext.IPAddress("yyyy")
	var ips map[ipResource]ipAddress

	BeforeEach(func() {
		stopChannel = make(chan struct{})

		// Create the mock K8s client.
		k8sClient = testclient.NewSimpleClientset()
		crdClient := fake.NewSimpleClientset()
		istioCrdClient := istio_fake.NewSimpleClientset()
		multiClusterCrdClient := multiCluster_fake.NewSimpleClientset()
		ingress = tests.NewIngressFixture()

		// Create a `k8scontext` to start listening to ingress resources.
		k8scontext.IsNetworkingV1PackageSupported = true
		ctxt = k8scontext.NewContext(k8sClient, crdClient, multiClusterCrdClient, istioCrdClient, []string{tests.Namespace}, 1000*time.Second, metricstore.NewFakeMetricStore())

		_, err := k8sClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
		Expect(err).Should(BeNil(), "Unable to create the namespace %s: %v", tests.Name, err)

		// create ingress in namespace
		_, err = k8sClient.NetworkingV1().Ingresses(tests.Namespace).Create(context.TODO(), ingress, metav1.CreateOptions{})
		Expect(err).Should(BeNil(), "Unabled to create ingress resource due to: %v", err)

		Expect(ctxt).ShouldNot(BeNil(), "Unable to create `k8scontext`")

		appGw = fixtures.GetAppGateway()
		newConfs := append(*appGw.FrontendIPConfigurations, fixtures.GetPrivateIPConfiguration())
		appGw.FrontendIPConfigurations = &newConfs
		controller = &AppGwIngressController{
			k8sContext: ctxt,
			ipAddressMap: map[string]k8scontext.IPAddress{
				*fixtures.GetPublicIPConfiguration().ID:  publicIP,
				*fixtures.GetPrivateIPConfiguration().ID: privateIP,
			},
			recorder: record.NewFakeRecorder(100),
		}
		cbCtx = &appgw.ConfigBuilderContext{
			IngressList: []*networking.Ingress{
				ingress,
			},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		ips = map[ipResource]ipAddress{"PublicIP": "xxxx", "PrivateIP": "yyyy"}
	})

	AfterEach(func() {
		close(stopChannel)
	})

	Context("test updateIngressStatus", func() {
		It("ensure that updateIngressStatus adds ipAddress to ingress", func() {
			controller.updateIngressStatus(&appGw, cbCtx, ingress, ips)
			updatedIngress, _ := k8sClient.NetworkingV1().Ingresses(ingress.Namespace).Get(context.TODO(), ingress.Name, metav1.GetOptions{})
			Expect(updatedIngress.Status.LoadBalancer.Ingress).Should(ContainElement(v1.LoadBalancerIngress{
				Hostname: "",
				IP:       string(publicIP),
			}))
			Expect(len(updatedIngress.Status.LoadBalancer.Ingress)).To(Equal(1))
		})

		It("ensure that updateIngressStatus adds private ipAddress when annotation is present", func() {
			ingress.Annotations[annotations.UsePrivateIPKey] = "true"
			updatedIngress, _ := k8sClient.NetworkingV1().Ingresses(ingress.Namespace).Update(context.TODO(), ingress, metav1.UpdateOptions{})
			Expect(annotations.UsePrivateIP(updatedIngress)).To(BeTrue())

			controller.updateIngressStatus(&appGw, cbCtx, ingress, ips)

			updatedIngress, _ = k8sClient.NetworkingV1().Ingresses(ingress.Namespace).Get(context.TODO(), ingress.Name, metav1.GetOptions{})
			Expect(updatedIngress.Status.LoadBalancer.Ingress).Should(ContainElement(v1.LoadBalancerIngress{
				Hostname: "",
				IP:       string(privateIP),
			}))
			Expect(len(updatedIngress.Status.LoadBalancer.Ingress)).To(Equal(1))
		})
	})

	Context("test ResetAllIngress", func() {
		It("ensure that ResetAllIngress sets removes the loadbalancer from ingress", func() {
			// Setup Ip Address first
			controller.updateIngressStatus(&appGw, cbCtx, ingress, ips)
			updatedIngress, _ := k8sClient.NetworkingV1().Ingresses(ingress.Namespace).Get(context.TODO(), ingress.Name, metav1.GetOptions{})
			Expect(updatedIngress.Status.LoadBalancer.Ingress).Should(ContainElement(v1.LoadBalancerIngress{
				Hostname: "",
				IP:       string(publicIP),
			}))

			// Reset should clear it
			controller.ResetAllIngress(&appGw, cbCtx)
			updatedIngress, _ = k8sClient.NetworkingV1().Ingresses(ingress.Namespace).Get(context.TODO(), ingress.Name, metav1.GetOptions{})
			Expect(len(updatedIngress.Status.LoadBalancer.Ingress)).To(Equal(0))
		})
	})
})
