// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package functional_tests

import (
	"flag"
	"testing"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
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
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"

	. "github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
)

func TestFunctional(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Appgw Suite")
}

var _ = Describe("Tests `appgw.ConfigBuilder`", func() {
	var stopChannel chan struct{}
	var ctxt *k8scontext.Context
	var configBuilder ConfigBuilder

	version.Version = "a"
	version.GitCommit = "b"
	version.BuildDate = "c"

	ingressNS := "test-ingress-controller"

	serviceName := "hello-world"

	// Frontend and Backend port.
	servicePort := Port(80)
	backendName := "http"
	backendPort := Port(1356)

	// Endpoints
	endpoint1 := "1.1.1.1"
	endpoint2 := "1.1.1.2"
	endpoint3 := "1.1.1.3"

	// Create the "test-ingress-controller" namespace.
	// We will create all our resources under this namespace.
	ns := &v1.Namespace{
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
			Namespace: ingressNS,
			Name:      tests.Name,
		},
	}

	// TODO(draychev): Get this from test fixtures -- tests.NewServiceFixture()
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ingressNS,
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

	serviceList := []*v1.Service{service}

	// Ideally we should be creating the `pods` resource instead of the `endpoints` resource
	// and allowing the k8s API server to create the `endpoints` resource which we end up consuming.
	// However since we are using a fake k8s client the resources are dumb which forces us to create the final
	// expected resource manually.
	endpoints := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ingressNS,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{IP: endpoint1},
					{IP: endpoint2},
					{IP: endpoint3},
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

	pod := tests.NewPodFixture(serviceName, ingressNS, backendName, int32(backendPort))

	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", "3")

	appGwIdentifier := Identifier{
		SubscriptionID: tests.Subscription,
		ResourceGroup:  tests.ResourceGroup,
		AppGwName:      tests.AppGwName,
	}

	BeforeEach(func() {
		stopChannel = make(chan struct{})

		// Create the mock K8s client.
		k8sClient := testclient.NewSimpleClientset()
		_, _ = k8sClient.CoreV1().Namespaces().Create(ns)
		_, _ = k8sClient.CoreV1().Nodes().Create(node)
		_, _ = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Create(ingress)
		_, _ = k8sClient.CoreV1().Services(ingressNS).Create(service)
		_, _ = k8sClient.CoreV1().Endpoints(ingressNS).Create(endpoints)
		_, _ = k8sClient.CoreV1().Pods(ingressNS).Create(pod)

		crdClient := fake.NewSimpleClientset()
		istioCrdClient := istio_fake.NewSimpleClientset()
		ctxt = k8scontext.NewContext(k8sClient, crdClient, istioCrdClient, []string{ingressNS}, 1000*time.Second)

		appGwy := &n.ApplicationGateway{
			ApplicationGatewayPropertiesFormat: NewAppGwyConfigFixture(),
		}

		configBuilder = NewConfigBuilder(ctxt, &appGwIdentifier, appGwy, record.NewFakeRecorder(100))
	})

	AfterEach(func() {
		close(stopChannel)
	})

	Context("Tests Application Gateway config creation", func() {
	ingress_A := &v1beta1.Ingress{
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
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
			Namespace: ingressNS,
			Name:      tests.Name,
		},
	}

	ingress_B := &v1beta1.Ingress{
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
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
			Namespace: ingressNS,
			Name:      tests.Name,
		},
	}
		It("Should have created correct App Gateway config JSON blob", func() {
			cbCtx := &ConfigBuilderContext{
				IngressList:  []*v1beta1.Ingress{
					ingress,
					ingress_A,
					ingress_B,
				},
				ServiceList:  serviceList,
				EnvVariables: environment.GetFakeEnv(),
			}
			two_ingresses(ctxt, stopChannel, cbCtx, configBuilder)
		})
	})
})
