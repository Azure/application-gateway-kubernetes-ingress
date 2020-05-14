// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
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
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

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
		ctx := context.TODO()
		options := metav1.CreateOptions{}
		_, _ = k8sClient.CoreV1().Namespaces().Create(ctx, ns, options)
		_, _ = k8sClient.CoreV1().Nodes().Create(ctx, node, options)
		_, _ = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Create(ctx, ingress, options)
		_, _ = k8sClient.CoreV1().Services(ingressNS).Create(ctx, service, options)
		_, _ = k8sClient.CoreV1().Endpoints(ingressNS).Create(ctx, endpoints, options)
		_, _ = k8sClient.CoreV1().Pods(ingressNS).Create(ctx, pod, options)

		crdClient := fake.NewSimpleClientset()
		istioCrdClient := istio_fake.NewSimpleClientset()
		ctxt = k8scontext.NewContext(k8sClient, crdClient, istioCrdClient, []string{ingressNS}, 1000*time.Second, metricstore.NewFakeMetricStore())

		appGwy := &n.ApplicationGateway{
			ApplicationGatewayPropertiesFormat: NewAppGwyConfigFixture(),
		}

		configBuilder = NewConfigBuilder(ctxt, &appGwIdentifier, appGwy, record.NewFakeRecorder(100), mocks.Clock{})
	})

	AfterEach(func() {
		close(stopChannel)
	})

	Context("Tests Application Gateway config creation", func() {
		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress},
			ServiceList:           serviceList,
			EnvVariables:          environment.GetFakeEnv(),
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		It("Should have created correct App Gateway config JSON blob", func() {
			// Start the informers. This will sync the cache with the latest ingress.
			err := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(err).ToNot(HaveOccurred())

			appGW, err := configBuilder.Build(cbCtx)
			Expect(err).ToNot(HaveOccurred())

			jsonBlob, err := appGW.MarshalJSON()
			Expect(err).ToNot(HaveOccurred())

			var into map[string]interface{}
			err = json.Unmarshal(jsonBlob, &into)
			Expect(err).ToNot(HaveOccurred())

			jsonBlob, err = json.MarshalIndent(into, "--", "    ")
			Expect(err).ToNot(HaveOccurred())

			jsonTxt := string(jsonBlob)

			expected := `{
--    "properties": {
--        "backendAddressPools": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/defaultaddresspool",
--                "name": "defaultaddresspool",
--                "properties": {
--                    "backendAddresses": []
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/pool-test-ingress-controller-hello-world-80-bp-80",
--                "name": "pool-test-ingress-controller-hello-world-80-bp-80",
--                "properties": {
--                    "backendAddresses": [
--                        {
--                            "ipAddress": "1.1.1.1"
--                        },
--                        {
--                            "ipAddress": "1.1.1.2"
--                        },
--                        {
--                            "ipAddress": "1.1.1.3"
--                        }
--                    ]
--                }
--            }
--        ],
--        "backendHttpSettingsCollection": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp-test-ingress-controller-hello-world-80-80---name--",
--                "name": "bp-test-ingress-controller-hello-world-80-80---name--",
--                "properties": {
--                    "cookieBasedAffinity": "Disabled",
--                    "pickHostNameFromBackendAddress": false,
--                    "port": 80,
--                    "probe": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/pb-test-ingress-controller-hello-world-80---name--"
--                    },
--                    "protocol": "Http",
--                    "requestTimeout": 30
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/defaulthttpsetting",
--                "name": "defaulthttpsetting",
--                "properties": {
--                    "cookieBasedAffinity": "Disabled",
--                    "pickHostNameFromBackendAddress": false,
--                    "port": 80,
--                    "probe": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http"
--                    },
--                    "protocol": "Http",
--                    "requestTimeout": 30
--                }
--            }
--        ],
--        "frontendIPConfigurations": [
--            {
--                "id": "--front-end-ip-id-1--",
--                "name": "xx3",
--                "properties": {
--                    "publicIPAddress": {
--                        "id": "xyz"
--                    }
--                }
--            },
--            {
--                "id": "--front-end-ip-id-2--",
--                "name": "yy3",
--                "properties": {
--                    "privateIPAddress": "abc"
--                }
--            }
--        ],
--        "frontendPorts": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontendPorts/fp-80",
--                "name": "fp-80",
--                "properties": {
--                    "port": 80
--                }
--            }
--        ],
--        "httpListeners": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-08ba2bb7da9df5927d900fca8ce96ba5",
--                "name": "fl-08ba2bb7da9df5927d900fca8ce96ba5",
--                "properties": {
--                    "frontendIPConfiguration": {
--                        "id": "--front-end-ip-id-1--"
--                    },
--                    "frontendPort": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontendPorts/fp-80"
--                    },
--                    "hostnames": [
--                        "foo.baz"
--                    ],
--                    "protocol": "Http",
--                    "requireServerNameIndication": false
--                }
--            }
--        ],
--        "probes": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http",
--                "name": "defaultprobe-Http",
--                "properties": {
--                    "host": "localhost",
--                    "interval": 30,
--                    "match": {},
--                    "minServers": 0,
--                    "path": "/",
--                    "pickHostNameFromBackendHttpSettings": false,
--                    "protocol": "Http",
--                    "timeout": 30,
--                    "unhealthyThreshold": 3
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Https",
--                "name": "defaultprobe-Https",
--                "properties": {
--                    "host": "localhost",
--                    "interval": 30,
--                    "match": {},
--                    "minServers": 0,
--                    "path": "/",
--                    "pickHostNameFromBackendHttpSettings": false,
--                    "protocol": "Https",
--                    "timeout": 30,
--                    "unhealthyThreshold": 3
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/pb-test-ingress-controller-hello-world-80---name--",
--                "name": "pb-test-ingress-controller-hello-world-80---name--",
--                "properties": {
--                    "host": "foo.baz",
--                    "interval": 30,
--                    "match": {},
--                    "minServers": 0,
--                    "path": "/",
--                    "pickHostNameFromBackendHttpSettings": false,
--                    "protocol": "Http",
--                    "timeout": 30,
--                    "unhealthyThreshold": 3
--                }
--            }
--        ],
--        "redirectConfigurations": [],
--        "requestRoutingRules": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/requestRoutingRules/rr-08ba2bb7da9df5927d900fca8ce96ba5",
--                "name": "rr-08ba2bb7da9df5927d900fca8ce96ba5",
--                "properties": {
--                    "backendAddressPool": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/pool-test-ingress-controller-hello-world-80-bp-80"
--                    },
--                    "backendHttpSettings": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp-test-ingress-controller-hello-world-80-80---name--"
--                    },
--                    "httpListener": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-08ba2bb7da9df5927d900fca8ce96ba5"
--                    },
--                    "ruleType": "Basic"
--                }
--            }
--        ],
--        "sku": {
--            "capacity": 3,
--            "name": "Standard_v2",
--            "tier": "Standard_v2"
--        },
--        "sslCertificates": [],
--        "urlPathMaps": []
--    },
--    "tags": {
--        "ingress-for-aks-cluster-id": "/subscriptions/subid/resourcegroups/aksresgp/providers/Microsoft.ContainerService/managedClusters/aksname",
--        "last-updated-by-k8s-ingress": "2009-11-17 20:34:58.651387237 +0000 UTC",
--        "managed-by-k8s-ingress": "a/b/c"
--    }
--}`

			linesAct := strings.Split(jsonTxt, "\n")
			linesExp := strings.Split(expected, "\n")

			Expect(len(linesAct)).To(Equal(len(linesExp)), "Line counts are different: ", len(linesAct), " vs ", len(linesExp), "\nActual:", jsonTxt, "\nExpected:", expected)

			for idx, line := range linesAct {
				curatedLineAct := strings.Trim(line, " ")
				curatedLineExp := strings.Trim(linesExp[idx], " ")
				Expect(curatedLineAct).To(Equal(curatedLineExp), fmt.Sprintf("Lines at index %d are different:\n%s\nvs expected:\n%s\nActual JSON:\n%s\n", idx, curatedLineAct, curatedLineExp, jsonTxt))
			}

		})
	})

	Context("Tests Application Gateway config creation with the SIMPLEST possible K8s YAML", func() {
		ingressNoRules := &v1beta1.Ingress{
			Spec: v1beta1.IngressSpec{
				Backend: &v1beta1.IngressBackend{
					ServiceName: serviceName,
					ServicePort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 80,
					},
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				},
				Name:      serviceName,
				Namespace: ingressNS,
			},
		}

		cbCtx := &ConfigBuilderContext{
			IngressList: []*v1beta1.Ingress{
				// ingress,
				ingressNoRules,
			},
			ServiceList:           serviceList,
			EnvVariables:          environment.GetFakeEnv(),
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		It("Should have created correct App Gateway config JSON blob", func() {
			// Start the informers. This will sync the cache with the latest ingress.
			err := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(err).ToNot(HaveOccurred())

			appGW, err := configBuilder.Build(cbCtx)
			Expect(err).ToNot(HaveOccurred())

			jsonBlob, err := appGW.MarshalJSON()
			Expect(err).ToNot(HaveOccurred())

			var into map[string]interface{}
			err = json.Unmarshal(jsonBlob, &into)
			Expect(err).ToNot(HaveOccurred())

			jsonBlob, err = json.MarshalIndent(into, "--", "    ")
			Expect(err).ToNot(HaveOccurred())

			jsonTxt := string(jsonBlob)

			expected := `{
--    "properties": {
--        "backendAddressPools": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/defaultaddresspool",
--                "name": "defaultaddresspool",
--                "properties": {
--                    "backendAddresses": []
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/pool-test-ingress-controller-hello-world-80-bp-80",
--                "name": "pool-test-ingress-controller-hello-world-80-bp-80",
--                "properties": {
--                    "backendAddresses": [
--                        {
--                            "ipAddress": "1.1.1.1"
--                        },
--                        {
--                            "ipAddress": "1.1.1.2"
--                        },
--                        {
--                            "ipAddress": "1.1.1.3"
--                        }
--                    ]
--                }
--            }
--        ],
--        "backendHttpSettingsCollection": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp-test-ingress-controller-hello-world-80-80-hello-world",
--                "name": "bp-test-ingress-controller-hello-world-80-80-hello-world",
--                "properties": {
--                    "cookieBasedAffinity": "Disabled",
--                    "pickHostNameFromBackendAddress": false,
--                    "port": 80,
--                    "probe": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http"
--                    },
--                    "protocol": "Http",
--                    "requestTimeout": 30
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/defaulthttpsetting",
--                "name": "defaulthttpsetting",
--                "properties": {
--                    "cookieBasedAffinity": "Disabled",
--                    "pickHostNameFromBackendAddress": false,
--                    "port": 80,
--                    "probe": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http"
--                    },
--                    "protocol": "Http",
--                    "requestTimeout": 30
--                }
--            }
--        ],
--        "frontendIPConfigurations": [
--            {
--                "id": "--front-end-ip-id-1--",
--                "name": "xx3",
--                "properties": {
--                    "publicIPAddress": {
--                        "id": "xyz"
--                    }
--                }
--            },
--            {
--                "id": "--front-end-ip-id-2--",
--                "name": "yy3",
--                "properties": {
--                    "privateIPAddress": "abc"
--                }
--            }
--        ],
--        "frontendPorts": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontendPorts/fp-80",
--                "name": "fp-80",
--                "properties": {
--                    "port": 80
--                }
--            }
--        ],
--        "httpListeners": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-e1903c8aa3446b7b3207aec6d6ecba8a",
--                "name": "fl-e1903c8aa3446b7b3207aec6d6ecba8a",
--                "properties": {
--                    "frontendIPConfiguration": {
--                        "id": "--front-end-ip-id-1--"
--                    },
--                    "frontendPort": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontendPorts/fp-80"
--                    },
--                    "hostnames": [],
--                    "protocol": "Http",
--                    "requireServerNameIndication": false
--                }
--            }
--        ],
--        "probes": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http",
--                "name": "defaultprobe-Http",
--                "properties": {
--                    "host": "localhost",
--                    "interval": 30,
--                    "match": {},
--                    "minServers": 0,
--                    "path": "/",
--                    "pickHostNameFromBackendHttpSettings": false,
--                    "protocol": "Http",
--                    "timeout": 30,
--                    "unhealthyThreshold": 3
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Https",
--                "name": "defaultprobe-Https",
--                "properties": {
--                    "host": "localhost",
--                    "interval": 30,
--                    "match": {},
--                    "minServers": 0,
--                    "path": "/",
--                    "pickHostNameFromBackendHttpSettings": false,
--                    "protocol": "Https",
--                    "timeout": 30,
--                    "unhealthyThreshold": 3
--                }
--            }
--        ],
--        "redirectConfigurations": [],
--        "requestRoutingRules": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/requestRoutingRules/rr-e1903c8aa3446b7b3207aec6d6ecba8a",
--                "name": "rr-e1903c8aa3446b7b3207aec6d6ecba8a",
--                "properties": {
--                    "backendAddressPool": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/pool-test-ingress-controller-hello-world-80-bp-80"
--                    },
--                    "backendHttpSettings": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp-test-ingress-controller-hello-world-80-80-hello-world"
--                    },
--                    "httpListener": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-e1903c8aa3446b7b3207aec6d6ecba8a"
--                    },
--                    "ruleType": "Basic"
--                }
--            }
--        ],
--        "sku": {
--            "capacity": 3,
--            "name": "Standard_v2",
--            "tier": "Standard_v2"
--        },
--        "sslCertificates": [],
--        "urlPathMaps": []
--    },
--    "tags": {
--        "ingress-for-aks-cluster-id": "/subscriptions/subid/resourcegroups/aksresgp/providers/Microsoft.ContainerService/managedClusters/aksname",
--        "last-updated-by-k8s-ingress": "2009-11-17 20:34:58.651387237 +0000 UTC",
--        "managed-by-k8s-ingress": "a/b/c"
--    }
--}`

			linesAct := strings.Split(jsonTxt, "\n")
			linesExp := strings.Split(expected, "\n")

			Expect(len(linesAct)).To(Equal(len(linesExp)), "Line counts are different: ", len(linesAct), " vs ", len(linesExp), "\nActual:", jsonTxt, "\nExpected:", expected)

			for idx, line := range linesAct {
				curatedLineAct := strings.Trim(line, " ")
				curatedLineExp := strings.Trim(linesExp[idx], " ")
				Expect(curatedLineAct).To(Equal(curatedLineExp), fmt.Sprintf("Lines at index %d are different:\n%s\nvs expected:\n%s\nActual JSON:\n%s\n", idx, curatedLineAct, curatedLineExp, jsonTxt))
			}

		})
	})

	Context("Tests Application Gateway config creation with hostname extensions", func() {
		ingressWithHostNameExtension := &v1beta1.Ingress{
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
					annotations.IngressClassKey:      annotations.ApplicationGatewayIngressClass,
					annotations.SslRedirectKey:       "true",
					annotations.HostNameExtensionKey: "bye.com, test.com",
				},
				Namespace: ingressNS,
				Name:      tests.Name,
			},
		}

		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingressWithHostNameExtension},
			ServiceList:           serviceList,
			EnvVariables:          environment.GetFakeEnv(),
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		It("Should have created correct App Gateway config JSON blob", func() {
			// Start the informers. This will sync the cache with the latest ingress.
			err := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(err).ToNot(HaveOccurred())

			appGW, err := configBuilder.Build(cbCtx)
			Expect(err).ToNot(HaveOccurred())

			jsonBlob, err := appGW.MarshalJSON()
			Expect(err).ToNot(HaveOccurred())

			var into map[string]interface{}
			err = json.Unmarshal(jsonBlob, &into)
			Expect(err).ToNot(HaveOccurred())

			jsonBlob, err = json.MarshalIndent(into, "--", "    ")
			Expect(err).ToNot(HaveOccurred())

			jsonTxt := string(jsonBlob)

			expected := `{
--    "properties": {
--        "backendAddressPools": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/defaultaddresspool",
--                "name": "defaultaddresspool",
--                "properties": {
--                    "backendAddresses": []
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/pool-test-ingress-controller-hello-world-80-bp-80",
--                "name": "pool-test-ingress-controller-hello-world-80-bp-80",
--                "properties": {
--                    "backendAddresses": [
--                        {
--                            "ipAddress": "1.1.1.1"
--                        },
--                        {
--                            "ipAddress": "1.1.1.2"
--                        },
--                        {
--                            "ipAddress": "1.1.1.3"
--                        }
--                    ]
--                }
--            }
--        ],
--        "backendHttpSettingsCollection": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp-test-ingress-controller-hello-world-80-80---name--",
--                "name": "bp-test-ingress-controller-hello-world-80-80---name--",
--                "properties": {
--                    "cookieBasedAffinity": "Disabled",
--                    "pickHostNameFromBackendAddress": false,
--                    "port": 80,
--                    "probe": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/pb-test-ingress-controller-hello-world-80---name--"
--                    },
--                    "protocol": "Http",
--                    "requestTimeout": 30
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/defaulthttpsetting",
--                "name": "defaulthttpsetting",
--                "properties": {
--                    "cookieBasedAffinity": "Disabled",
--                    "pickHostNameFromBackendAddress": false,
--                    "port": 80,
--                    "probe": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http"
--                    },
--                    "protocol": "Http",
--                    "requestTimeout": 30
--                }
--            }
--        ],
--        "frontendIPConfigurations": [
--            {
--                "id": "--front-end-ip-id-1--",
--                "name": "xx3",
--                "properties": {
--                    "publicIPAddress": {
--                        "id": "xyz"
--                    }
--                }
--            },
--            {
--                "id": "--front-end-ip-id-2--",
--                "name": "yy3",
--                "properties": {
--                    "privateIPAddress": "abc"
--                }
--            }
--        ],
--        "frontendPorts": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontendPorts/fp-80",
--                "name": "fp-80",
--                "properties": {
--                    "port": 80
--                }
--            }
--        ],
--        "httpListeners": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-b2dba5a02f61d8615cc23d8df120ac20",
--                "name": "fl-b2dba5a02f61d8615cc23d8df120ac20",
--                "properties": {
--                    "frontendIPConfiguration": {
--                        "id": "--front-end-ip-id-1--"
--                    },
--                    "frontendPort": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontendPorts/fp-80"
--                    },
--                    "hostnames": [
--                        "foo.baz",
--                        "bye.com",
--                        "test.com"
--                    ],
--                    "protocol": "Http",
--                    "requireServerNameIndication": false
--                }
--            }
--        ],
--        "probes": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http",
--                "name": "defaultprobe-Http",
--                "properties": {
--                    "host": "localhost",
--                    "interval": 30,
--                    "match": {},
--                    "minServers": 0,
--                    "path": "/",
--                    "pickHostNameFromBackendHttpSettings": false,
--                    "protocol": "Http",
--                    "timeout": 30,
--                    "unhealthyThreshold": 3
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Https",
--                "name": "defaultprobe-Https",
--                "properties": {
--                    "host": "localhost",
--                    "interval": 30,
--                    "match": {},
--                    "minServers": 0,
--                    "path": "/",
--                    "pickHostNameFromBackendHttpSettings": false,
--                    "protocol": "Https",
--                    "timeout": 30,
--                    "unhealthyThreshold": 3
--                }
--            },
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/pb-test-ingress-controller-hello-world-80---name--",
--                "name": "pb-test-ingress-controller-hello-world-80---name--",
--                "properties": {
--                    "host": "foo.baz",
--                    "interval": 30,
--                    "match": {},
--                    "minServers": 0,
--                    "path": "/",
--                    "pickHostNameFromBackendHttpSettings": false,
--                    "protocol": "Http",
--                    "timeout": 30,
--                    "unhealthyThreshold": 3
--                }
--            }
--        ],
--        "redirectConfigurations": [],
--        "requestRoutingRules": [
--            {
--                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/requestRoutingRules/rr-b2dba5a02f61d8615cc23d8df120ac20",
--                "name": "rr-b2dba5a02f61d8615cc23d8df120ac20",
--                "properties": {
--                    "backendAddressPool": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/pool-test-ingress-controller-hello-world-80-bp-80"
--                    },
--                    "backendHttpSettings": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp-test-ingress-controller-hello-world-80-80---name--"
--                    },
--                    "httpListener": {
--                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-b2dba5a02f61d8615cc23d8df120ac20"
--                    },
--                    "ruleType": "Basic"
--                }
--            }
--        ],
--        "sku": {
--            "capacity": 3,
--            "name": "Standard_v2",
--            "tier": "Standard_v2"
--        },
--        "sslCertificates": [],
--        "urlPathMaps": []
--    },
--    "tags": {
--        "ingress-for-aks-cluster-id": "/subscriptions/subid/resourcegroups/aksresgp/providers/Microsoft.ContainerService/managedClusters/aksname",
--        "last-updated-by-k8s-ingress": "2009-11-17 20:34:58.651387237 +0000 UTC",
--        "managed-by-k8s-ingress": "a/b/c"
--    }
--}`

			linesAct := strings.Split(jsonTxt, "\n")
			linesExp := strings.Split(expected, "\n")

			Expect(len(linesAct)).To(Equal(len(linesExp)), "Line counts are different: ", len(linesAct), " vs ", len(linesExp), "\nActual:", jsonTxt, "\nExpected:", expected)

			for idx, line := range linesAct {
				curatedLineAct := strings.Trim(line, " ")
				curatedLineExp := strings.Trim(linesExp[idx], " ")
				Expect(curatedLineAct).To(Equal(curatedLineExp), fmt.Sprintf("Lines at index %d are different:\n%s\nvs expected:\n%s\nActual JSON:\n%s\n", idx, curatedLineAct, curatedLineExp, jsonTxt))
			}

		})
	})
})
