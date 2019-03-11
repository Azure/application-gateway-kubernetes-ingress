package appgw

import (
	go_flag "flag"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

type appGWSettingsChecker struct {
	total   int                                               // Number of expected sub-resources of this setting.
	checker func(*network.ApplicationGatewayPropertiesFormat) // A method to test the values with this setting. Will be run if the checker is not null.
}

type appGwConfigSettings struct {
	backendHTTPSettingsCollection appGWSettingsChecker // Number of backend HTTP settings.
	backendAddressPools           appGWSettingsChecker // Number of backend address pool.
	hTTPListeners                 appGWSettingsChecker // Number of HTTP Listeners
	requestRoutingRules           appGWSettingsChecker // Number of routing rules.
	uRLPathMaps                   appGWSettingsChecker // Number of URL path maps.
}

var _ = Describe("Tests `appgw.ConfigBuilder`", func() {
	var k8sClient kubernetes.Interface
	var ctxt *k8scontext.Context
	var configBuilder ConfigBuilder

	domainName := "hello.com"
	ingressNS := "test-ingress-controller"
	ingressName := "hello-world"
	serviceName := "hello-world"

	// Frontend and Backend port.
	servicePort := int32(80)
	backendPort := int32(1356)

	// Endpoints
	endpoint1 := "1.1.1.1"
	endpoint2 := "1.1.1.2"
	endpoint3 := "1.1.1.3"

	// Paths
	hiPath := "/hi"

	// Create the "test-ingress-controller" namespace.
	// We will create all our resources under this namespace.
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ingressNS,
		},
	}

	// Create the Ingress resource.
	ingress := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: ingressNS,
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: domainName,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: hiPath,
									Backend: v1beta1.IngressBackend{
										ServiceName: serviceName,
										ServicePort: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: int32(servicePort),
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
			Name:      serviceName,
			Namespace: ingressNS,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "frontendPort",
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: backendPort,
					},
					Protocol: v1.ProtocolTCP,
					Port:     servicePort,
				},
			},
			Selector: map[string]string{"app": "frontend"},
		},
	}

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
					{
						IP: endpoint1,
					},
					{
						IP: endpoint2,
					},
					{
						IP: endpoint3,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     "frontend",
						Port:     backendPort,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	go_flag.Lookup("logtostderr").Value.Set("true")
	go_flag.Set("v", "3")

	// Method to test all the ingress that have been added to the K8s context.
	testIngress := func() []*v1beta1.Ingress {
		// Get all the ingresses
		ingressList := ctxt.GetHTTPIngressList()
		// There should be only one ingress
		Expect(len(ingressList)).To(Equal(1), "Expected only one ingress resource but got: %d", len(ingressList))
		// Make sure it is the ingress we stored.
		Expect(ingressList[0]).To(Equal(ingress))

		return ingressList
	}

	defaultBackendHTTPSettingsChecker := func(appGW *network.ApplicationGatewayPropertiesFormat) {
		expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
		httpSettingsName := generateHTTPSettingsName(generateBackendID(ingress, expectedBackend).serviceFullName(), fmt.Sprintf("%d", servicePort), backendPort, ingress.Name)
		httpSettings := &network.ApplicationGatewayBackendHTTPSettings{
			Etag: to.StringPtr("*"),
			Name: &httpSettingsName,
			ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
				Protocol: network.HTTP,
				Port:     &backendPort,
				Path:     to.StringPtr(""),
			},
		}

		// Test the default backend HTTP settings.
		Expect((*appGW.BackendHTTPSettingsCollection)[0]).To(Equal(defaultBackendHTTPSettings()))
		// Test the ingress backend HTTP setting that we installed.
		Expect((*appGW.BackendHTTPSettingsCollection)[1]).To(Equal(*httpSettings))
	}

	defaultBackendAddressPoolChecker := func(appGW *network.ApplicationGatewayPropertiesFormat) {
		expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
		addressPoolName := generateAddressPoolName(generateBackendID(ingress, expectedBackend).serviceFullName(), fmt.Sprintf("%d", servicePort), backendPort)
		addressPoolAddresses := [](network.ApplicationGatewayBackendAddress){{IPAddress: &endpoint1}, {IPAddress: &endpoint2}, {IPAddress: &endpoint3}}

		addressPool := &network.ApplicationGatewayBackendAddressPool{
			Etag: to.StringPtr("*"),
			Name: &addressPoolName,
			ApplicationGatewayBackendAddressPoolPropertiesFormat: &network.ApplicationGatewayBackendAddressPoolPropertiesFormat{
				BackendAddresses: &addressPoolAddresses,
			},
		}

		// Test the default backend address pool.
		Expect((*appGW.BackendAddressPools)[0]).To(Equal(defaultBackendAddressPool()))
		// Test the ingress backend address pool that we installed.
		Expect((*appGW.BackendAddressPools)[1]).To(Equal(*addressPool))
	}

	defaultHTTPListenersChecker := func(appGW *network.ApplicationGatewayPropertiesFormat) {
		// Test the listener.
		appGwIdentifier := Identifier{}
		frontendPortID := appGwIdentifier.frontendPortID(generateFrontendPortName(80))
		httpListenerName := generateHTTPListenerName(frontendListenerIdentifier{80, domainName})
		httpListener := &network.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: &httpListenerName,
			ApplicationGatewayHTTPListenerPropertiesFormat: &network.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration: resourceRef("*"),
				FrontendPort:            resourceRef(frontendPortID),
				Protocol:                network.HTTP,
				HostName:                &domainName,
			},
		}

		Expect((*appGW.HTTPListeners)[0]).To(Equal(*httpListener))
	}

	defaultRequestRoutingRulesChecker := func(appGW *network.ApplicationGatewayPropertiesFormat) {
		Expect(*((*appGW.RequestRoutingRules)[0].Name)).To(Equal(generateRequestRoutingRuleName(frontendListenerIdentifier{80, domainName})))
		Expect((*appGW.RequestRoutingRules)[0].RuleType).To(Equal(network.PathBasedRouting))
	}

	defaultURLPathMapsChecker := func(appGW *network.ApplicationGatewayPropertiesFormat) {
		Expect(*((*appGW.URLPathMaps)[0].Name)).To(Equal(generateURLPathMapName(frontendListenerIdentifier{80, domainName})))
		// Check the `pathRule` stored within the `urlPathMap`.
		Expect(len(*((*appGW.URLPathMaps)[0].PathRules))).To(Equal(1), "Expected one path based rule, but got: %d", len(*((*appGW.URLPathMaps)[0].PathRules)))

		pathRule := (*((*appGW.URLPathMaps)[0].PathRules))[0]
		Expect(len(*(pathRule.Paths))).To(Equal(1), "Expected a single path in path-based rules, but got: %d", len(*(pathRule.Paths)))
		// Check the exact path that was set.
		Expect((*pathRule.Paths)[0]).To(Equal("/hi"))
	}

	testAGConfig := func(ingressList []*v1beta1.Ingress, settings appGwConfigSettings) {
		// Add HTTP settings.
		configBuilder, err := configBuilder.BackendHTTPSettingsCollection(ingressList)
		Expect(err).Should(BeNil(), "Error in generating the HTTP Settings: %v", err)

		// Retrieve the implementation of the `ConfigBuilder` interface.
		appGW := configBuilder.Build()
		// We will have a default HTTP setting that gets added, and an HTTP setting corresponding to port `backendPort`
		Expect(len(*appGW.BackendHTTPSettingsCollection)).To(Equal(settings.backendHTTPSettingsCollection.total), "Did not find expected number of backend HTTP settings")

		// Test the value of the backend HTTP settings if the checker has been setup.
		if settings.backendHTTPSettingsCollection.checker != nil {
			settings.backendHTTPSettingsCollection.checker(appGW)
		}

		// Add backend address pools. We need the HTTP settings before we can add the backend address pools.
		configBuilder, err = configBuilder.BackendAddressPools(ingressList)
		Expect(err).Should(BeNil(), "Error in generating the backend address pools: %v", err)

		// Retrieve the implementation of the `ConfigBuilder` interface.
		appGW = configBuilder.Build()
		// We will have a default backend address pool that gets added, and a backend pool corresponding to our service.
		Expect(len(*appGW.BackendAddressPools)).To(Equal(settings.backendAddressPools.total), "Did not find expected number of backend address pool.")

		if settings.backendAddressPools.checker != nil {
			settings.backendAddressPools.checker(appGW)
		}

		// Add the listeners. We need the backend address pools before we can add HTTP listeners.
		configBuilder, err = configBuilder.HTTPListeners(ingressList)
		Expect(err).Should(BeNil(), "Error in generating the HTTP listeners: %v", err)

		// Retrieve the implementation of the `ConfigBuilder` interface.
		appGW = configBuilder.Build()
		// Ingress allows listeners on port 80 or port 443. Therefore in this particular case we would have only a single listener
		Expect(len(*appGW.HTTPListeners)).To(Equal(settings.hTTPListeners.total), "Did not find expected number of HTTP listeners")

		if settings.hTTPListeners.checker != nil {
			settings.hTTPListeners.checker(appGW)
		}

		// RequestRoutingRules depends on the previous operations
		configBuilder, err = configBuilder.RequestRoutingRules(ingressList)
		Expect(err).Should(BeNil(), "Error in generating the routing rules: %v", err)

		// Retrieve the implementation of the `ConfigBuilder` interface.
		appGW = configBuilder.Build()
		Expect(len(*appGW.RequestRoutingRules)).To(Equal(settings.requestRoutingRules.total), "Did not find expected number of request routing rules")

		if settings.requestRoutingRules.checker != nil {
			settings.requestRoutingRules.checker(appGW)
		}

		// Check the `urlPathMaps`
		Expect(len(*appGW.URLPathMaps)).To(Equal(settings.uRLPathMaps.total), "Did not find expected number of URL path maps")
		if settings.uRLPathMaps.checker != nil {
			settings.uRLPathMaps.checker(appGW)
		}
	}

	BeforeEach(func() {
		// Create the mock K8s client.
		k8sClient = testclient.NewSimpleClientset()

		_, err := k8sClient.CoreV1().Namespaces().Create(ns)
		Expect(err).Should(BeNil(), "Unable to create the namespace %s: %v", ingressNS, err)

		_, err = k8sClient.Extensions().Ingresses(ingressNS).Create(ingress)
		Expect(err).Should(BeNil(), "Unabled to create ingress resource due to: %v", err)

		// Create the service.
		_, err = k8sClient.CoreV1().Services(ingressNS).Create(service)
		Expect(err).Should(BeNil(), "Unabled to create service resource due to: %v", err)

		// Create the endpoints associated with this service.
		_, err = k8sClient.CoreV1().Endpoints(ingressNS).Create(endpoints)
		Expect(err).Should(BeNil(), "Unabled to create endpoints resource due to: %v", err)

		// Create a `k8scontext` to start listiening to ingress resources.
		ctxt = k8scontext.NewContext(k8sClient, ingressNS, 1000*time.Second)
		Expect(ctxt).ShouldNot(BeNil(), "Unable to create `k8scontext`")

		// Initialize the `ConfigBuilder`
		configBuilder = NewConfigBuilder(ctxt, &Identifier{}, &network.ApplicationGatewayPropertiesFormat{})

		builder, ok := configBuilder.(*appGwConfigBuilder)
		Expect(ok).Should(BeTrue(), "Unable to get the more specific configBuilder implementation")

		// Since this is a mock the `Application Gateway v2` does not have a public IP. During configuration process
		// the controller would expect the `Application Gateway v2` to have some public IP before it starts generating
		// configuration for the application gateway, hence creating this dummy configuration in the application gateway configuration.
		builder.appGwConfig.FrontendIPConfigurations = &[]network.ApplicationGatewayFrontendIPConfiguration{
			{
				Name: to.StringPtr("*"),
				Etag: to.StringPtr("*"),
				ID:   to.StringPtr("*"),
			},
		}
	})

	Context("Tests Application Gateway Configuration", func() {
		It("Should be able to create Application Gateway Configuration from Ingress", func() {
			ctxt.Run()

			// Start the informers. This will sync the cache with the latest ingress.
			ingressList := testIngress()

			testAGConfig(ingressList, appGwConfigSettings{
				backendHTTPSettingsCollection: appGWSettingsChecker{
					total:   2,
					checker: defaultBackendHTTPSettingsChecker,
				},
				backendAddressPools: appGWSettingsChecker{
					total:   2,
					checker: defaultBackendAddressPoolChecker,
				},
				hTTPListeners: appGWSettingsChecker{
					total:   1,
					checker: defaultHTTPListenersChecker,
				},
				requestRoutingRules: appGWSettingsChecker{
					total:   1,
					checker: defaultRequestRoutingRulesChecker,
				},
				uRLPathMaps: appGWSettingsChecker{
					total:   1,
					checker: defaultURLPathMapsChecker,
				},
			})
		})
	})

	Context("Tests Ingress Controller when Service doesn't exists", func() {
		It("Should be able to create Application Gateway Configuration from Ingress with empty backend pool.", func() {
			// Delete the service
			options := &metav1.DeleteOptions{}
			err := k8sClient.CoreV1().Services(ingressNS).Delete(serviceName, options)
			Expect(err).Should(BeNil(), "Unable to delete service resource due to: %v", err)

			// Delete the Endpoint
			err = k8sClient.CoreV1().Endpoints(ingressNS).Delete(serviceName, options)
			Expect(err).Should(BeNil(), "Unable to delete endpoint resource due to: %v", err)

			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run()

			// Get all the ingresses
			ingressList := testIngress()

			EmptyBackendHTTPSettingsChecker := func(appGW *network.ApplicationGatewayPropertiesFormat) {
				expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
				httpSettingsName := generateHTTPSettingsName(generateBackendID(ingress, expectedBackend).serviceFullName(), fmt.Sprintf("%d", servicePort), servicePort, ingress.Name)
				httpSettings := &network.ApplicationGatewayBackendHTTPSettings{
					Etag: to.StringPtr("*"),
					Name: &httpSettingsName,
					ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
						Protocol: network.HTTP,
						Port:     &servicePort,
						Path:     to.StringPtr(""),
					},
				}

				// Test the default backend HTTP settings.
				Expect((*appGW.BackendHTTPSettingsCollection)[0]).To(Equal(defaultBackendHTTPSettings()))
				// Test the ingress backend HTTP setting that we installed.
				Expect((*appGW.BackendHTTPSettingsCollection)[1]).To(Equal(*httpSettings))
			}

			EmptyBackendAddressPoolChecker := func(appGW *network.ApplicationGatewayPropertiesFormat) {
				// Test the default backend address pool.
				Expect((*appGW.BackendAddressPools)[0]).To(Equal(defaultBackendAddressPool()))
			}

			testAGConfig(ingressList, appGwConfigSettings{
				backendHTTPSettingsCollection: appGWSettingsChecker{
					total:   2,
					checker: EmptyBackendHTTPSettingsChecker,
				},
				backendAddressPools: appGWSettingsChecker{
					total:   1,
					checker: EmptyBackendAddressPoolChecker,
				},
				hTTPListeners: appGWSettingsChecker{
					total:   1,
					checker: defaultHTTPListenersChecker,
				},
				requestRoutingRules: appGWSettingsChecker{
					total:   1,
					checker: defaultRequestRoutingRulesChecker,
				},
				uRLPathMaps: appGWSettingsChecker{
					total:   1,
					checker: defaultURLPathMapsChecker,
				},
			})

		})
	})
	Context("Tests Ingress Controller Annotations", func() {
		It("Should be able to create Application Gateway Configuration from Ingress with backend prefix.", func() {
			ingress, err := k8sClient.Extensions().Ingresses(ingressNS).Get(ingressName, metav1.GetOptions{})
			Expect(err).Should(BeNil(), "Unabled to create ingress resource due to: %v", err)

			// Set the ingress annotation for this ingress.
			ingress.Annotations[annotations.BackendPathPrefixKey] = "/test"

			// Update the ingress.
			_, err = k8sClient.Extensions().Ingresses(ingressNS).Update(ingress)
			Expect(err).Should(BeNil(), "Unabled to update ingress resource due to: %v", err)

			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run()

			// Method to test all the ingress that have been added to the K8s context.
			backendPrefixIngress := func() []*v1beta1.Ingress {
				// Get all the ingresses
				ingressList := ctxt.GetHTTPIngressList()
				// There should be only one ingress
				Expect(len(ingressList)).To(Equal(1), "Expected only one ingress resource but got: %d", len(ingressList))
				// Make sure it is the ingress we stored.
				Expect(ingressList[0]).To(Equal(ingress))

				return ingressList
			}

			// Get all the ingresses
			ingressList := backendPrefixIngress()

			backendPrefixHTTPSettingsChecker := func(appGW *network.ApplicationGatewayPropertiesFormat) {
				expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
				httpSettingsName := generateHTTPSettingsName(generateBackendID(ingress, expectedBackend).serviceFullName(), fmt.Sprintf("%d", servicePort), backendPort, ingress.Name)
				httpSettings := &network.ApplicationGatewayBackendHTTPSettings{
					Etag: to.StringPtr("*"),
					Name: &httpSettingsName,
					ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
						Protocol: network.HTTP,
						Port:     &backendPort,
						Path:     to.StringPtr("/test"),
					},
				}

				// Test the default backend HTTP settings.
				Expect((*appGW.BackendHTTPSettingsCollection)[0]).To(Equal(defaultBackendHTTPSettings()))
				// Test the ingress backend HTTP setting that we installed.
				Expect((*appGW.BackendHTTPSettingsCollection)[1]).To(Equal(*httpSettings))
			}

			testAGConfig(ingressList, appGwConfigSettings{
				backendHTTPSettingsCollection: appGWSettingsChecker{
					total:   2,
					checker: backendPrefixHTTPSettingsChecker,
				},
				backendAddressPools: appGWSettingsChecker{
					total:   2,
					checker: defaultBackendAddressPoolChecker,
				},
				hTTPListeners: appGWSettingsChecker{
					total:   1,
					checker: defaultHTTPListenersChecker,
				},
				requestRoutingRules: appGWSettingsChecker{
					total:   1,
					checker: defaultRequestRoutingRulesChecker,
				},
				uRLPathMaps: appGWSettingsChecker{
					total:   1,
					checker: defaultURLPathMapsChecker,
				},
			})

		})
	})

})
