// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
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
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

var _ = Describe("Tests `appgw.ConfigBuilder`", func() {
	version.Version = "a"
	version.GitCommit = "b"
	version.BuildDate = "c"

	ingressNS := "test-ingressPrivateIP-controller"

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
			Namespace: tests.Namespace,
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
			Namespace: tests.Namespace,
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
	_, err := k8sClient.CoreV1().Namespaces().Create(nameSpace)
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.CoreV1().Nodes().Create(node)
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Create(ingressPublicIP)
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Create(ingressPrivateIP)
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.CoreV1().Services(ingressNS).Create(service)
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.CoreV1().Endpoints(ingressNS).Create(endpoints)
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.CoreV1().Pods(ingressNS).Create(pod1)
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	_, err = k8sClient.CoreV1().Pods(ingressNS).Create(pod2)
	It("should have not failed", func() { Expect(err).ToNot(HaveOccurred()) })

	crdClient := fake.NewSimpleClientset()
	istioCrdClient := istio_fake.NewSimpleClientset()
	ctxt := k8scontext.NewContext(k8sClient, crdClient, istioCrdClient, []string{ingressNS}, 1000*time.Second)

	appGwy := &n.ApplicationGateway{}
	// Since this is a mock the `Application Gateway v2` does not have a public IP. During configuration process
	// the controller would expect the `Application Gateway v2` to have some public IP before it starts generating
	// configuration for the application gateway, hence creating this dummy configuration in the application gateway configuration.
	appGwy.ApplicationGatewayPropertiesFormat = &n.ApplicationGatewayPropertiesFormat{
		FrontendIPConfigurations: &[]n.ApplicationGatewayFrontendIPConfiguration{
			{
				Name: to.StringPtr("--public-ip--"),
				Etag: to.StringPtr("*"),
				ID:   to.StringPtr("--public--ip--id--"),
				ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &n.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
					PublicIPAddress: &n.SubResource{
						ID: to.StringPtr("x/y/z"),
					},
				},
			},
			{
				Name: to.StringPtr("--private--ip--"),
				Etag: to.StringPtr("*"),
				ID:   to.StringPtr("--private--ip--id--"),
				ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &n.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
					PrivateIPAddress:          to.StringPtr("10.1.1.1"),
					PrivateIPAllocationMethod: "Static",
				},
			},
		},
	}

	// Initialize the `ConfigBuilder`
	configBuilder := NewConfigBuilder(ctxt, &appGwIdentifier, appGwy, record.NewFakeRecorder(100))

	Context("Tests Application Gateway config creation", func() {
		cbCtx := &ConfigBuilderContext{
			IngressList: []*v1beta1.Ingress{
				ingressPrivateIP,
				ingressPublicIP,
			},
			ServiceList:  serviceList,
			EnvVariables: environment.GetFakeEnv(),
		}

		It("Should have created correct App Gateway config JSON blob", func() {
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
    --            }
    --        ],
    --        "backendHttpSettingsCollection": [
    --            {
    --                "etag": "*",
    --                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp---namespace-----service-name---443-443-external-ingress-resource",
    --                "name": "bp---namespace-----service-name---443-443-external-ingress-resource",
    --                "properties": {
    --                    "port": 443,
    --                    "probe": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http"
    --                    },
    --                    "protocol": "Http"
    --                }
    --            },
    --            {
    --                "etag": "*",
    --                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp---namespace-----service-name---80-80-internal-ingress-resource",
    --                "name": "bp---namespace-----service-name---80-80-internal-ingress-resource",
    --                "properties": {
    --                    "port": 80,
    --                    "probe": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http"
    --                    },
    --                    "protocol": "Http"
    --                }
    --            },
    --            {
    --                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/defaulthttpsetting",
    --                "name": "defaulthttpsetting",
    --                "properties": {
    --                    "port": 80,
    --                    "probe": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/probes/defaultprobe-Http"
    --                    },
    --                    "protocol": "Http"
    --                }
    --            }
    --        ],
    --        "frontendIPConfigurations": [
    --            {
    --                "etag": "*",
    --                "id": "--public--ip--id--",
    --                "name": "--public-ip--",
    --                "properties": {
    --                    "publicIPAddress": {
    --                        "id": "x/y/z"
    --                    }
    --                }
    --            },
    --            {
    --                "etag": "*",
    --                "id": "--private--ip--id--",
    --                "name": "--private--ip--",
    --                "properties": {
    --                    "privateIPAddress": "10.1.1.1",
    --                    "privateIPAllocationMethod": "Static"
    --                }
    --            }
    --        ],
    --        "frontendPorts": [
    --            {
    --                "etag": "*",
    --                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontEndPorts/fp-80",
    --                "name": "fp-80",
    --                "properties": {
    --                    "port": 80
    --                }
    --            }
    --        ],
    --        "httpListeners": [
    --            {
    --                "etag": "*",
    --                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-80",
    --                "name": "fl-80",
    --                "properties": {
    --                    "frontendIPConfiguration": {
    --                        "id": "--public--ip--id--"
    --                    },
    --                    "frontendPort": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontEndPorts/fp-80"
    --                    },
    --                    "hostName": "",
    --                    "protocol": "Http"
    --                }
    --            },
    --            {
    --                "etag": "*",
    --                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-80-privateip",
    --                "name": "fl-80-privateip",
    --                "properties": {
    --                    "frontendIPConfiguration": {
    --                        "id": "--private--ip--id--"
    --                    },
    --                    "frontendPort": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontEndPorts/fp-80"
    --                    },
    --                    "hostName": "",
    --                    "protocol": "Http"
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
    --                    "path": "/",
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
    --                    "path": "/",
    --                    "protocol": "Https",
    --                    "timeout": 30,
    --                    "unhealthyThreshold": 3
    --                }
    --            }
    --        ],
    --        "redirectConfigurations": null,
    --        "requestRoutingRules": [
    --            {
    --                "etag": "*",
    --                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/requestRoutingRules/rr-80",
    --                "name": "rr-80",
    --                "properties": {
    --                    "backendAddressPool": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/defaultaddresspool"
    --                    },
    --                    "backendHttpSettings": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp---namespace-----service-name---80-80-internal-ingress-resource"
    --                    },
    --                    "httpListener": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-80-privateip"
    --                    },
    --                    "ruleType": "Basic"
    --                }
    --            },
    --            {
    --                "etag": "*",
    --                "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/requestRoutingRules/rr-80",
    --                "name": "rr-80",
    --                "properties": {
    --                    "backendAddressPool": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendAddressPools/defaultaddresspool"
    --                    },
    --                    "backendHttpSettings": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/backendHttpSettingsCollection/bp---namespace-----service-name---443-443-external-ingress-resource"
    --                    },
    --                    "httpListener": {
    --                        "id": "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/httpListeners/fl-80"
    --                    },
    --                    "ruleType": "Basic"
    --                }
    --            }
    --        ],
    --        "sslCertificates": null,
    --        "urlPathMaps": null
    --    },
    --    "tags": {
    --        "ingress-for-aks-cluster-id": "/subscriptions/subid/resourcegroups/aksresgp/providers/Microsoft.ContainerService/managedClusters/aksname",
    --        "managed-by-k8s-ingress": "a/b/c"
    --    }
    --}`

			linesAct := strings.Split(jsonTxt, "\n")
			linesExp := strings.Split(expected, "\n")

			Expect(len(linesAct)).To(Equal(len(linesExp)), fmt.Sprintf("Line counts are different: actual=%d vs expected=%d\nActual:%s\nExpected:%s", len(linesAct), len(linesExp), jsonTxt, expected))

			for idx, line := range linesAct {
				curatedLineAct := strings.Trim(line, " ")
				curatedLineExp := strings.Trim(linesExp[idx], " ")
				Expect(curatedLineAct).To(Equal(curatedLineExp), fmt.Sprintf("Lines at index %d are different:\n%s\nvs expected:\n%s\nActual JSON:\n%s\n", idx, curatedLineAct, curatedLineExp, jsonTxt))
			}

		})
	})
})
