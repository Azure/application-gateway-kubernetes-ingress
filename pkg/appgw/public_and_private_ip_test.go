// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"context"
	"flag"
	"fmt"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	istio_fake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/mocks"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

var _ = Describe("Tests `appgw.ConfigBuilder`", func() {
	version.Version = "a"
	version.GitCommit = "b"
	version.BuildDate = "c"

	ingressNS := tests.Namespace

	// Create the "test-ingressPrivateIP-controller" namespace.
	// We will create all our resources under this namespace.
	nameSpace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ingressNS,
		},
	}

	// Create a node
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Spec: v1.NodeSpec{
			ProviderID: "azure:///subscriptions/subid/resourceGroups/MC_aksresgp_aksname_location/providers/Microsoft.Compute/virtualMachines/vmname",
		},
	}

	ingressPublicIP := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
			Namespace: ingressNS,
			Name:      "external-ingress-resource",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/*",
									Backend: v1beta1.IngressBackend{
										ServiceName: tests.ServiceName,
										ServicePort: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 443,
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []v1beta1.IngressTLS{
				{
					Hosts: []string{
						"pub.lic",
						"www.contoso.com",
						"ftp.contoso.com",
						tests.Host,
						"",
					},
					SecretName: tests.NameOfSecret,
				},
				{
					Hosts:      []string{},
					SecretName: tests.NameOfSecret,
				},
			},
		},
	}

	// Create the Ingress resource.
	ingressPrivateIP := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				annotations.UsePrivateIPKey: "true",
			},
			Namespace: ingressNS,
			Name:      "internal-ingress-resource",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/*",
									Backend: v1beta1.IngressBackend{
										ServiceName: tests.ServiceName,
										ServicePort: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tests.ServiceName,
			Namespace: ingressNS,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "http",
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 80,
					},
					Protocol: v1.ProtocolTCP,
					Port:     int32(80),
				},
				{
					Name: "https",
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 80,
					},
					Protocol: v1.ProtocolTCP,
					Port:     int32(443),
				},
			},
			Selector: map[string]string{"app": "web--app--name"},
		},
	}

	serviceList := []*v1.Service{
		service,
	}

	endpoints := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tests.ServiceName,
			Namespace: ingressNS,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{IP: "1.1.1.1"},
					{IP: "1.1.1.2"},
					{IP: "1.1.1.3"},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "http",
						Port:     int32(80),
						Protocol: v1.ProtocolTCP,
					},
					{
						Name:     "https",
						Port:     int32(443),
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	pod1 := tests.NewPodFixture("pod1", ingressNS, "http", int32(80))
	pod2 := tests.NewPodFixture("pod2", ingressNS, "https", int32(80))

	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", "3")

	appGwIdentifier := Identifier{
		SubscriptionID: tests.Subscription,
		ResourceGroup:  tests.ResourceGroup,
		AppGwName:      tests.AppGwName,
	}

	// Create the mock K8s client.
	k8sClient := testclient.NewSimpleClientset()

	It("should have not failed", func() {
		_, err := k8sClient.CoreV1().Namespaces().Create(context.TODO(), nameSpace, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	crdClient := fake.NewSimpleClientset()
	istioCrdClient := istio_fake.NewSimpleClientset()
	ctxt := k8scontext.NewContext(k8sClient, crdClient, istioCrdClient, []string{ingressNS}, 1000*time.Second, metricstore.NewFakeMetricStore())

	secret := tests.NewSecretTestFixture()

	err := ctxt.Caches.Secret.Add(secret)
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	secKey := utils.GetResourceKey(secret.Namespace, secret.Name)

	err = ctxt.CertificateSecretStore.ConvertSecret(secKey, secret)
	It("should have converted the certificate", func() { Expect(err).ToNot(HaveOccurred()) })

	pfx := ctxt.CertificateSecretStore.GetPfxCertificate(secKey)
	It("should have found the pfx certificate", func() { Expect(pfx).ToNot(BeNil()) })

	ctxtSecret := ctxt.GetSecret(secKey)
	It("should have found the secret", func() { Expect(ctxtSecret).To(Equal(secret)) })

	_, err = k8sClient.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Create(context.TODO(), ingressPublicIP, metav1.CreateOptions{})
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Update(context.TODO(), ingressPublicIP, metav1.UpdateOptions{})
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred(), "Unable to update ingress resource due to: %v", err) })

	_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Create(context.TODO(), ingressPrivateIP, metav1.CreateOptions{})
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Update(context.TODO(), ingressPrivateIP, metav1.UpdateOptions{})
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred(), "Unable to update ingress resource due to: %v", err) })

	_, err = k8sClient.CoreV1().Services(ingressNS).Create(context.TODO(), service, metav1.CreateOptions{})
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.CoreV1().Endpoints(ingressNS).Create(context.TODO(), endpoints, metav1.CreateOptions{})
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.CoreV1().Pods(ingressNS).Create(context.TODO(), pod1, metav1.CreateOptions{})
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.CoreV1().Pods(ingressNS).Create(context.TODO(), pod2, metav1.CreateOptions{})
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	appGwy := &n.ApplicationGateway{
		ApplicationGatewayPropertiesFormat: NewAppGwyConfigFixture(),
	}

	// Initialize the `ConfigBuilder`
	configBuilder := NewConfigBuilder(ctxt, &appGwIdentifier, appGwy, record.NewFakeRecorder(100), mocks.Clock{})

	Context("Tests Application Gateway config creation", func() {
		cbCtx := &ConfigBuilderContext{
			IngressList: []*v1beta1.Ingress{
				ingressPrivateIP,
				ingressPublicIP,
			},
			ServiceList:           serviceList,
			EnvVariables:          environment.GetFakeEnv(),
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		It("Should have created correct App Gateway config JSON blob", func() {
			appGW, err := configBuilder.Build(cbCtx)
			Expect(err).ToNot(HaveOccurred())

			jsonBlob, err := appGW.MarshalJSON()
			Expect(err).ToNot(HaveOccurred())

			Expect(appGW.HTTPListeners).ToNot(BeNil())

			foundPrivateIPListener := false
			for _, listener := range *appGW.HTTPListeners {
				if *listener.FrontendPort.ID == "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontendPorts/fp-80" {
					foundPrivateIPListener = true
					Expect(*listener.FrontendIPConfiguration.ID).To(Equal("--front-end-ip-id-2--"), fmt.Sprintf("Expecting to find private IP frontend configuration attached here."))
				}
			}

			Expect(foundPrivateIPListener).To(BeTrue(), fmt.Sprintf("Expecting to find a listener using private IP. Actual JSON:\n%s\n", string(jsonBlob)))
		})
	})
})
