// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build unittest

package functests

import (
	"context"
	"flag"
	"testing"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	. "github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	istio_fake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/mocks"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

func TestFunctional(t *testing.T) {
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Appgw Suite")
}

var _ = ginkgo.Describe("Tests `appgw.ConfigBuilder`", func() {
	var stopChannel chan struct{}
	var ctxt *k8scontext.Context
	var configBuilder ConfigBuilder

	version.Version = "a"
	version.GitCommit = "b"
	version.BuildDate = "c"

	serviceName := "hello-world"
	serviceNameA := "hello-world-a"
	serviceNameB := "hello-world-b"
	serviceNameC := "hello-world-c"

	serviceNameHttps := "hello-world-https"

	// Frontend and Backend port.
	servicePort := Port(80)
	backendName := "http"
	backendPort := Port(1356)
	httpsBackendName := "https"
	httpsServicePort := Port(443)

	// Create the "test-ingress-controller" namespace.
	// We will create all our resources under this namespace.
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: tests.Namespace,
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

	// Create the Ingress resource.
	ingress := &v1beta1.Ingress{
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "foo.baz",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1beta1.IngressBackend{
										ServiceName: serviceName,
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
			TLS: []v1beta1.IngressTLS{
				{
					Hosts: []string{
						"foo.baz",
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
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				annotations.SslRedirectKey:  "true",
			},
			Namespace: tests.Namespace,
			Name:      tests.Name,
		},
	}

	ingressSecret := tests.NewSecretTestFixture()

	// Create an Ingress resource for the same domain but no TLS
	ingressFooBazNoTLS := &v1beta1.Ingress{
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "foo.baz",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/.well-known/acme-challenge/blahBlahBBLLAAHH",
									Backend: v1beta1.IngressBackend{
										ServiceName: serviceNameB,
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
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
			Namespace: tests.Namespace,
			Name:      tests.Name + "FooBazNoTLS",
		},
	}

	ingressFooBazNoTLSHostNameFromAnnotation := &v1beta1.Ingress{
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/.well-known/acme-challenge/blahBlahBBLLAAHH",
									Backend: v1beta1.IngressBackend{
										ServiceName: serviceNameB,
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
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey:      annotations.ApplicationGatewayIngressClass,
				annotations.HostNameExtensionKey: "foo.baz",
			},
			Namespace: tests.Namespace,
			Name:      tests.Name + "FooBazNoTLSHostNameFromAnnotation",
		},
	}

	ingressOtherNamespace := &v1beta1.Ingress{
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "foo.baz",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/b",
									Backend: v1beta1.IngressBackend{
										ServiceName: serviceNameC,
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
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
			Namespace: tests.OtherNamespace,
			Name:      tests.Name + "OtherNamespace",
		},
	}

	// TODO(draychev): Get this from test fixtures -- tests.NewServiceFixture()
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: tests.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "servicePort",
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: backendName,
					},
					Protocol: v1.ProtocolTCP,
					Port:     int32(servicePort),
				},
			},
			Selector: map[string]string{"app": "frontend"},
		},
	}

	serviceA := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameA,
			Namespace: tests.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "servicePort",
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: backendName,
					},
					Protocol: v1.ProtocolTCP,
					Port:     int32(servicePort),
				},
			},
			Selector: map[string]string{"app": "frontend"},
		},
	}

	serviceB := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameB,
			Namespace: tests.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "servicePort",
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: backendName,
					},
					Protocol: v1.ProtocolTCP,
					Port:     int32(servicePort),
				},
			},
			Selector: map[string]string{"app": "frontend"},
		},
	}

	serviceC := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameC,
			Namespace: tests.OtherNamespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "servicePort",
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: backendName,
					},
					Protocol: v1.ProtocolTCP,
					Port:     int32(servicePort),
				},
			},
			Selector: map[string]string{"app": "frontend"},
		},
	}

	serviceHttps := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameHttps,
			Namespace: tests.HTTPSBackendNamespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "serviceHttpsPort",
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: httpsBackendName,
					},
					Protocol: v1.ProtocolTCP,
					Port:     int32(httpsServicePort),
				},
			},
			Selector: map[string]string{"app": "frontend"},
		},
	}

	serviceList := []*v1.Service{
		service,
		serviceA,
		serviceB,
		serviceHttps,
	}

	// Ideally we should be creating the `pods` resource instead of the `endpoints` resource
	// and allowing the k8s API server to create the `endpoints` resource which we end up consuming.
	// However since we are using a fake k8s client the resources are dumb which forces us to create the final
	// expected resource manually.
	endpoints := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: tests.Namespace,
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
						Name:     "servicePort",
						Port:     int32(servicePort),
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	endpointsA := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameA,
			Namespace: tests.Namespace,
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
						Name:     "servicePort",
						Port:     int32(servicePort),
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	endpointsB := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameB,
			Namespace: tests.Namespace,
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
						Name:     "servicePort",
						Port:     int32(servicePort),
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	endpointsC := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameC,
			Namespace: tests.OtherNamespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{IP: "21.21.21.21"},
					{IP: "21.21.21.22"},
					{IP: "21.21.21.23"},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "servicePort",
						Port:     int32(servicePort),
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	endpointsHttps := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceNameHttps,
			Namespace: tests.HTTPSBackendNamespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{IP: "11.21.21.21"},
					{IP: "11.21.21.22"},
					{IP: "11.21.21.23"},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "serviceHttpsPort",
						Port:     int32(httpsServicePort),
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	pod := tests.NewPodFixture(serviceName, tests.Namespace, backendName, int32(backendPort))
	podB := tests.NewPodFixture(serviceNameB, tests.Namespace, backendName, int32(backendPort))
	podC := tests.NewPodFixture(serviceNameC, tests.OtherNamespace, backendName, int32(backendPort))
	podHttps := tests.NewPodHTTPSFixture(serviceNameHttps, tests.HTTPSBackendNamespace, httpsBackendName, int32(httpsServicePort))

	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", "3")

	appGwIdentifier := Identifier{
		SubscriptionID: tests.Subscription,
		ResourceGroup:  tests.ResourceGroup,
		AppGwName:      tests.AppGwName,
	}

	ginkgo.BeforeEach(func() {
		stopChannel = make(chan struct{})

		// Create the mock K8s client.
		k8sClient := testclient.NewSimpleClientset()
		_, _ = k8sClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
		_, _ = k8sClient.ExtensionsV1beta1().Ingresses(tests.Namespace).Create(context.TODO(), ingress, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), serviceA, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Services(tests.Namespace).Create(context.TODO(), serviceB, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Services(tests.HTTPSBackendNamespace).Create(context.TODO(), serviceHttps, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Services(tests.OtherNamespace).Create(context.TODO(), serviceC, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Endpoints(tests.Namespace).Create(context.TODO(), endpoints, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Endpoints(tests.Namespace).Create(context.TODO(), endpointsA, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Endpoints(tests.Namespace).Create(context.TODO(), endpointsB, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Endpoints(tests.HTTPSBackendNamespace).Create(context.TODO(), endpointsHttps, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Endpoints(tests.OtherNamespace).Create(context.TODO(), endpointsC, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Pods(tests.Namespace).Create(context.TODO(), podB, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Pods(tests.HTTPSBackendNamespace).Create(context.TODO(), podHttps, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Pods(tests.OtherNamespace).Create(context.TODO(), podC, metav1.CreateOptions{})
		_, _ = k8sClient.CoreV1().Secrets(tests.Namespace).Create(context.TODO(), ingressSecret, metav1.CreateOptions{})

		crdClient := fake.NewSimpleClientset()
		istioCrdClient := istio_fake.NewSimpleClientset()
		namespaces := []string{
			tests.Namespace,
			tests.OtherNamespace,
		}
		ctxt = k8scontext.NewContext(k8sClient, crdClient, istioCrdClient, namespaces, 1000*time.Second, metricstore.NewFakeMetricStore())

		secKey := utils.GetResourceKey(ingressSecret.Namespace, ingressSecret.Name)
		_ = ctxt.CertificateSecretStore.ConvertSecret(secKey, ingressSecret)
		_ = ctxt.CertificateSecretStore.GetPfxCertificate(secKey)

		appGwy := &n.ApplicationGateway{
			ApplicationGatewayPropertiesFormat: NewAppGwyConfigFixture(),
		}

		configBuilder = NewConfigBuilder(ctxt, &appGwIdentifier, appGwy, record.NewFakeRecorder(100), mocks.Clock{})
	})

	ginkgo.AfterEach(func() {
		close(stopChannel)
	})

	ginkgo.Context("Tests Application Gateway config creation", func() {
		ingressA := &v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						// This one has no host
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/A/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameA,
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				},
				Namespace: tests.Namespace,
				Name:      tests.Name + "A",
			},
		}

		ingressB := &v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						// This one has no host
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/B/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameB,
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				},
				Namespace: tests.Namespace,
				Name:      tests.Name + "B",
			},
		}

		ingressHttpsBackend := &v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						// This one has no host
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/A/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameHttps,
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
			},
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotations.IngressClassKey:             annotations.ApplicationGatewayIngressClass,
					annotations.AppGwSslCertificate:         "ssl-certificate",
					annotations.BackendProtocolKey:          "https",
					annotations.AppGwTrustedRootCertificate: "root-certificate",
					annotations.SslRedirectKey:              "true",
				},
				Namespace: tests.HTTPSBackendNamespace,
				Name:      tests.Name + "HttpsBackend",
			},
		}

		ingressBWithExtendedHostName := &v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						// This one has no host
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/B/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameB,
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotations.IngressClassKey:      annotations.ApplicationGatewayIngressClass,
					annotations.HostNameExtensionKey: "test.com, t*.com",
				},
				Namespace: tests.Namespace,
				Name:      tests.Name + "BWithExtendedHostName",
			},
		}

		ingressSlashNothing := &v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						// This one has no host
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameB,
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				},
				Namespace: tests.Namespace,
				Name:      tests.Name + "SlashNothing",
			},
		}

		ingressSlashNothingSlashSomething := &v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						// This one has no host
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameB,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 80,
											},
										},
									},
									{
										Path: "/A",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameA,
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				},
				Namespace: tests.Namespace,
				Name:      tests.Name + "SlashNothingSlashSomething",
			},
		}

		ingressMultiplePathRules := &v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Rules: []v1beta1.IngressRule{
					{
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/A/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameA,
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
					{
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/B/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameA,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 80,
											},
										},
									},
									{
										Path: "/C/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameA,
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
					{
						Host: "site.com",
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/A/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameA,
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
					{
						Host: "site.com",
						IngressRuleValue: v1beta1.IngressRuleValue{
							HTTP: &v1beta1.HTTPIngressRuleValue{
								Paths: []v1beta1.HTTPIngressPath{
									{
										Path: "/B/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameA,
											ServicePort: intstr.IntOrString{
												Type:   intstr.Int,
												IntVal: 80,
											},
										},
									},
									{
										Path: "/C/",
										Backend: v1beta1.IngressBackend{
											ServiceName: serviceNameA,
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
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				},
				Namespace: tests.Namespace,
				Name:      tests.Name + "MultiplePathRules",
			},
		}

		ginkgo.It("THREE Ingress Resources", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingress,
					ingressA,
					ingressB,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "three_ingresses.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("Https Backend Ingress Resources", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingressHttpsBackend,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "one_ingress_https_backend.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("Https Backend Ingress Resources without backend-protocol specified", func() {
			newAnnotation := map[string]string{
				annotations.IngressClassKey:     annotations.ApplicationGatewayIngressClass,
				annotations.AppGwSslCertificate: "ssl-certificate",
			}

			ingressHttpsBackend.SetAnnotations(newAnnotation)
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingressHttpsBackend,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			// protocol of httpSettings, probe should be https when backend port is at 443
			check(cbCtx, "one_ingress_https_backend_without_backend_protocol.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("ONE Ingress Resources with / (nothing) path", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingressSlashNothing,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "one_ingress_slash_nothing.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("ONE Ingress Resources with / (nothing), and /A/ path", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingressA,
					ingressSlashNothing,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "one_ingress_slash_slashnothing.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("ONE Ingress Resources with multiple paths rules", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingressMultiplePathRules,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "one_ingress_with_multiple_path_rules.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("TWO Ingress Resources, one with / another with /something paths", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingressSlashNothing,
					ingressA,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "two_ingresses_slash_slashsomething.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("TWO Ingress Resources for the same domain: one with TLS, another without", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingress,
					ingressFooBazNoTLS,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "two_ingresses_same_domain_tls_notls.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("TWO Ingress Resources same path and hostname but one has host in ingress rule and other has annotation", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingressFooBazNoTLS,
					ingressFooBazNoTLSHostNameFromAnnotation,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "two_ingresses_same_hostname_value_different_locations.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("TWO Ingress Resources with same path but one with extended hostname and one without", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingressBWithExtendedHostName,
					ingressA,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "two_ingresses_with_and_without_extended_hostname.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("Preexisting port w/ same port number", func() {

			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingress,
				},
				ServiceList:           serviceList,
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yyxx"),
				ExistingPortsByNumber: map[Port]n.ApplicationGatewayFrontendPort{
					Port(80):   fixtures.GetDefaultPort(),
					Port(8989): fixtures.GetPort(8989),
				},
			}
			check(cbCtx, "duplicate_ports.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("WAF Annotation", func() {
			annotatedIngress := ingressSlashNothingSlashSomething
			annotatedIngress.Annotations[annotations.FirewallPolicy] = "/some/policy/here"

			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					annotatedIngress,
				},
				ServiceList:  serviceList,
				EnvVariables: environment.GetFakeEnv(),
				ExistingPortsByNumber: map[Port]n.ApplicationGatewayFrontendPort{
					Port(80): fixtures.GetDefaultPort(),
				},
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "waf_annotation.json", stopChannel, ctxt, configBuilder)
		})

		ginkgo.It("Health Probes: same container labels; different namespaces", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ingress,
					ingressOtherNamespace,
				},
				ServiceList: []*v1.Service{
					serviceA,
					serviceC,
				},
				EnvVariables:          environment.GetFakeEnv(),
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			check(cbCtx, "health_probes_same_labels_different_namespaces.json", stopChannel, ctxt, configBuilder)
		})

	})
})
