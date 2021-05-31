// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure/tags"
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

type appGWSettingsChecker struct {
	total   int                                         // Number of expected sub-resources of this setting.
	checker func(*n.ApplicationGatewayPropertiesFormat) // A method to test the values with this setting. Will be run if the checker is not null.
}

type appGwConfigSettings struct {
	healthProbesCollection        appGWSettingsChecker // Number of health probes.
	backendHTTPSettingsCollection appGWSettingsChecker // Number of backend HTTP settings.
	backendAddressPools           appGWSettingsChecker // Number of backend address pool.
	listeners                     appGWSettingsChecker // Number of HTTP Listeners
	requestRoutingRules           appGWSettingsChecker // Number of routing rules.
	uRLPathMaps                   appGWSettingsChecker // Number of URL path maps.
}

var _ = Describe("Tests `appgw.ConfigBuilder`", func() {
	var k8sClient kubernetes.Interface
	var ctxt *k8scontext.Context
	var configBuilder ConfigBuilder
	var stopChannel chan struct{}
	var appGwIdentifier Identifier

	version.Version = "a"
	version.GitCommit = "b"
	version.BuildDate = "c"

	domainName := "hello.com"
	ingressNS := "test-ingress-controller"
	ingressName := "hello-world"
	serviceName := "hello-world"

	// Frontend and Backend port.
	servicePort := Port(80)
	backendName := "http"
	backendPort := Port(1356)

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

	serviceList := []*v1.Service{
		service,
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
						Name:     "servicePort",
						Port:     int32(backendPort),
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}

	pod := tests.NewPodFixture(serviceName, ingressNS, backendName, int32(backendPort))

	// Method to test all the ingress that have been added to the K8s context.
	testIngress := func() []*v1beta1.Ingress {
		// Get all the ingresses
		ingressList := ctxt.ListHTTPIngresses()
		// There should be only one ingress
		Expect(len(ingressList)).To(Equal(1), "Expected only one ingress resource but got: %d", len(ingressList))
		// Make sure it is the ingress we stored.
		Expect(ingressList[0]).To(Equal(ingress))

		return ingressList
	}

	defaultHealthProbesChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
		expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
		probeName := generateProbeName(expectedBackend.ServiceName, expectedBackend.ServicePort.String(), ingress)
		probe := &n.ApplicationGatewayProbe{
			Name: &probeName,
			ID:   to.StringPtr(appGwIdentifier.probeID(probeName)),
			ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            n.HTTP,
				Host:                                to.StringPtr(tests.Host),
				Path:                                to.StringPtr(tests.HealthPath),
				Interval:                            to.Int32Ptr(20),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				Timeout:                             to.Int32Ptr(5),
				Port:                                to.Int32Ptr(9090),
				Match:                               &n.ApplicationGatewayProbeHealthResponseMatch{},
				PickHostNameFromBackendHTTPSettings: to.BoolPtr(false),
				MinServers:                          to.Int32Ptr(0),
			},
		}

		probes := *appGW.Probes
		Expect(len(probes)).To(Equal(3))

		// Test the default health probe.
		Expect(probes).To(ContainElement(defaultProbe(appGwIdentifier, n.HTTP)))
		// Test the ingress health probe that we installed.
		Expect(probes).To(ContainElement(*probe))
	}

	defaultBackendHTTPSettingsChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
		expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
		probeID := appGwIdentifier.probeID(generateProbeName(expectedBackend.ServiceName, expectedBackend.ServicePort.String(), ingress))
		httpSettingsName := generateHTTPSettingsName(generateBackendID(ingress, nil, nil, expectedBackend).serviceFullName(), fmt.Sprintf("%d", servicePort), Port(backendPort), ingress.Name)
		httpSettings := &n.ApplicationGatewayBackendHTTPSettings{
			Etag: to.StringPtr("*"),
			Name: &httpSettingsName,
			ID:   to.StringPtr(appGwIdentifier.HTTPSettingsID(httpSettingsName)),
			ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
				Protocol:                       n.HTTP,
				Port:                           to.Int32Ptr(int32(backendPort)),
				Path:                           nil,
				HostName:                       nil,
				Probe:                          resourceRef(probeID),
				PickHostNameFromBackendAddress: to.BoolPtr(false),
				CookieBasedAffinity:            n.Disabled,
				RequestTimeout:                 to.Int32Ptr(30),
			},
		}

		// Test the default backend HTTP settings.
		Expect(*appGW.BackendHTTPSettingsCollection).To(ContainElement(defaultBackendHTTPSettings(appGwIdentifier, n.HTTP)))
		// Test the ingress backend HTTP setting that we installed.
		Expect(*appGW.BackendHTTPSettingsCollection).To(ContainElement(*httpSettings))
	}

	defaultBackendAddressPoolChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
		expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
		addressPoolName := generateAddressPoolName(generateBackendID(ingress, nil, nil, expectedBackend).serviceFullName(), fmt.Sprintf("%d", servicePort), Port(backendPort))
		addressPoolAddresses := []n.ApplicationGatewayBackendAddress{{IPAddress: &endpoint1}, {IPAddress: &endpoint2}, {IPAddress: &endpoint3}}

		addressPool := &n.ApplicationGatewayBackendAddressPool{
			Etag: to.StringPtr("*"),
			Name: &addressPoolName,
			ID:   to.StringPtr(appGwIdentifier.AddressPoolID(addressPoolName)),
			ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
				BackendAddresses: &addressPoolAddresses,
			},
		}

		// Test the default backend address pool.
		Expect(*appGW.BackendAddressPools).To(ContainElement(defaultBackendAddressPool(appGwIdentifier)))
		// Test the ingress backend address pool that we installed.
		Expect(*appGW.BackendAddressPools).To(ContainElement(*addressPool))
	}

	defaultListenersChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
		// Test the listener.
		frontendPortID := appGwIdentifier.frontendPortID(generateFrontendPortName(80))
		_, listenerName := newTestListenerID(Port(80), []string{domainName}, false)
		listener := &n.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: &listenerName,
			ID:   to.StringPtr(appGwIdentifier.listenerID(listenerName)),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration:     resourceRef("--front-end-ip-id-1--"),
				FrontendPort:                resourceRef(frontendPortID),
				Protocol:                    n.HTTP,
				HostNames:                   &[]string{domainName},
				RequireServerNameIndication: to.BoolPtr(false),
			},
		}

		Expect((*appGW.HTTPListeners)[0]).To(Equal(*listener))
	}

	baseRequestRoutingRulesChecker := func(appGW *n.ApplicationGatewayPropertiesFormat, frontEndPort Port, host string) {
		listenerID, _ := newTestListenerID(Port(frontEndPort), []string{host}, false)
		Expect(*((*appGW.RequestRoutingRules)[0].Name)).To(Equal(generateRequestRoutingRuleName(listenerID)))
		Expect((*appGW.RequestRoutingRules)[0].RuleType).To(Equal(n.PathBasedRouting))
	}

	defaultRequestRoutingRulesChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
		baseRequestRoutingRulesChecker(appGW, 80, domainName)
	}

	defaultHTTPSRequestRoutingRulesChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
		baseRequestRoutingRulesChecker(appGW, 443, domainName)
	}

	baseURLPathMapsChecker := func(appGW *n.ApplicationGatewayPropertiesFormat, frontEndPort Port, host string) {
		listenerID, _ := newTestListenerID(Port(frontEndPort), []string{host}, false)
		Expect(*((*appGW.URLPathMaps)[0].Name)).To(Equal(generateURLPathMapName(listenerID)))
		// Check the `pathRule` stored within the `urlPathMap`.
		Expect(len(*((*appGW.URLPathMaps)[0].PathRules))).To(Equal(1), "Expected one path based rule, but got: %d", len(*((*appGW.URLPathMaps)[0].PathRules)))

		pathRule := (*((*appGW.URLPathMaps)[0].PathRules))[0]
		Expect(len(*(pathRule.Paths))).To(Equal(1), "Expected a single path in path-based rules, but got: %d", len(*(pathRule.Paths)))
		// Check the exact path that was set.
		Expect((*pathRule.Paths)[0]).To(Equal("/hi"))
	}

	defaultURLPathMapsChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
		baseURLPathMapsChecker(appGW, 80, domainName)
	}

	defaultHTTPSURLPathMapsChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
		baseURLPathMapsChecker(appGW, 443, domainName)
	}

	testAGConfig := func(ingressList []*v1beta1.Ingress, serviceList []*v1.Service, settings appGwConfigSettings) {
		cbCtx := &ConfigBuilderContext{
			IngressList:           ingressList,
			ServiceList:           serviceList,
			EnvVariables:          environment.GetFakeEnv(),
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		appGW, err := configBuilder.Build(cbCtx)
		Ω(err).ToNot(HaveOccurred(), "Error in generating the Health Probes: %v", err)

		// We will have a default HTTP setting that gets added, and an HTTP setting corresponding to port `backendPort`
		Expect(len(*appGW.BackendHTTPSettingsCollection)).To(Equal(settings.backendHTTPSettingsCollection.total), "Did not find expected number of backend HTTP settings")

		// Test the value of the health probes if the checker has been setup.
		if settings.healthProbesCollection.checker != nil {
			settings.healthProbesCollection.checker(appGW.ApplicationGatewayPropertiesFormat)
		}

		// Test the value of the backend HTTP settings if the checker has been setup.
		if settings.backendHTTPSettingsCollection.checker != nil {
			settings.backendHTTPSettingsCollection.checker(appGW.ApplicationGatewayPropertiesFormat)
		}

		// We will have a default backend address pool that gets added, and a backend pool corresponding to our service.
		Expect(len(*appGW.BackendAddressPools)).To(Equal(settings.backendAddressPools.total), "Did not find expected number of backend address pool.")

		if settings.backendAddressPools.checker != nil {
			settings.backendAddressPools.checker(appGW.ApplicationGatewayPropertiesFormat)
		}

		// Ingress allows listeners on port 80 or port 443. Therefore in this particular case we would have only a single listener
		Expect(len(*appGW.HTTPListeners)).To(Equal(settings.listeners.total), "Did not find expected number of HTTP listeners")

		if settings.listeners.checker != nil {
			settings.listeners.checker(appGW.ApplicationGatewayPropertiesFormat)
		}

		Expect(len(*appGW.RequestRoutingRules)).To(Equal(settings.requestRoutingRules.total),
			fmt.Sprintf("Expected %d request routing rules; Got %d", settings.requestRoutingRules.total, len(*appGW.RequestRoutingRules)))

		if settings.requestRoutingRules.checker != nil {
			settings.requestRoutingRules.checker(appGW.ApplicationGatewayPropertiesFormat)
		}

		// Check the `urlPathMaps`
		Expect(len(*appGW.URLPathMaps)).To(Equal(settings.uRLPathMaps.total), "Did not find expected number of URL path maps")
		if settings.uRLPathMaps.checker != nil {
			settings.uRLPathMaps.checker(appGW.ApplicationGatewayPropertiesFormat)
		}

		// Check tags
		Expect(len(appGW.Tags)).To(Equal(2))
		expected := map[string]*string{
			tags.ManagedByK8sIngress:    to.StringPtr("a/b/c"),
			tags.IngressForAKSClusterID: to.StringPtr("/subscriptions/subid/resourcegroups/aksresgp/providers/Microsoft.ContainerService/managedClusters/aksname"),
		}
		Expect(appGW.Tags).To(Equal(expected))
	}

	ingressEvent := func() {
		for {
			select {
			case event := <-ctxt.Work:
				// Check if we got an event of type secret.
				if _, ok := event.Value.(*v1beta1.Ingress); ok {
					return
				}
			}
		}
	}

	BeforeEach(func() {
		stopChannel = make(chan struct{})
		appGwIdentifier = Identifier{
			SubscriptionID: tests.Subscription,
			ResourceGroup:  tests.ResourceGroup,
			AppGwName:      tests.AppGwName,
		}

		// Create the mock K8s client.
		k8sClient = testclient.NewSimpleClientset()

		_, err := k8sClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
		Ω(err).ToNot(HaveOccurred(), "Unable to create the namespace %s: %v", ingressNS, err)

		_, err = k8sClient.CoreV1().Nodes().Create(context.TODO(), node, metav1.CreateOptions{})
		Ω(err).ToNot(HaveOccurred(), "Unable to create node resource due to: %v", err)

		_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Create(context.TODO(), ingress, metav1.CreateOptions{})
		Ω(err).ToNot(HaveOccurred(), "Unable to create ingress resource due to: %v", err)

		// Create the service.
		_, err = k8sClient.CoreV1().Services(ingressNS).Create(context.TODO(), service, metav1.CreateOptions{})
		Ω(err).ToNot(HaveOccurred(), "Unable to create service resource due to: %v", err)

		// Create the endpoints associated with this service.
		_, err = k8sClient.CoreV1().Endpoints(ingressNS).Create(context.TODO(), endpoints, metav1.CreateOptions{})
		Ω(err).ToNot(HaveOccurred(), "Unable to create endpoints resource due to: %v", err)

		// Create the pods associated with this service.
		_, err = k8sClient.CoreV1().Pods(ingressNS).Create(context.TODO(), pod, metav1.CreateOptions{})
		Ω(err).ToNot(HaveOccurred(), "Unable to create pods resource due to: %v", err)

		// Create a mock CRD Client
		crdClient := fake.NewSimpleClientset()

		// Create a mock Istio CRD Client
		istioCrdClient := istio_fake.NewSimpleClientset()

		// Create a `k8scontext` to start listiening to ingress resources.

		ctxt = k8scontext.NewContext(k8sClient, crdClient, istioCrdClient, []string{ingressNS}, 1000*time.Second, metricstore.NewFakeMetricStore())
		Expect(ctxt).ShouldNot(BeNil(), "Unable to create `k8scontext`")

		// Initialize the `ConfigBuilder`
		appGw := &n.ApplicationGateway{
			ApplicationGatewayPropertiesFormat: NewAppGwyConfigFixture(),
		}
		configBuilder = NewConfigBuilder(ctxt, &appGwIdentifier, appGw, record.NewFakeRecorder(100), mocks.Clock{})

		_, ok := configBuilder.(*appGwConfigBuilder)
		Expect(ok).Should(BeTrue(), "Unable to get the more specific configBuilder implementation")
	})

	AfterEach(func() {
		close(stopChannel)
	})

	Context("Tests Application Gateway Configuration", func() {
		It("Should be able to create Application Gateway Configuration from Ingress", func() {
			// Start the informers. This will sync the cache with the latest ingress.
			err := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Ω(err).ToNot(HaveOccurred())

			// Wait for the controller to receive an ingress update.
			ingressEvent()

			ingressList := testIngress()

			testAGConfig(ingressList, serviceList, appGwConfigSettings{
				healthProbesCollection: appGWSettingsChecker{
					total:   2,
					checker: defaultHealthProbesChecker,
				},
				backendHTTPSettingsCollection: appGWSettingsChecker{
					total:   2,
					checker: defaultBackendHTTPSettingsChecker,
				},
				backendAddressPools: appGWSettingsChecker{
					total:   2,
					checker: defaultBackendAddressPoolChecker,
				},
				listeners: appGWSettingsChecker{
					total:   1,
					checker: defaultListenersChecker,
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
			options := metav1.DeleteOptions{}
			ctx := context.TODO()
			err := k8sClient.CoreV1().Services(ingressNS).Delete(ctx, serviceName, options)
			Ω(err).ToNot(HaveOccurred(), "Unable to delete service resource due to: %v", err)

			// Delete the Endpoint
			err = k8sClient.CoreV1().Endpoints(ingressNS).Delete(ctx, serviceName, options)
			Ω(err).ToNot(HaveOccurred(), "Unable to delete endpoint resource due to: %v", err)

			// Start the informers. This will sync the cache with the latest ingress.
			err = ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Ω(err).ToNot(HaveOccurred())

			// Wait for the controller to receive an ingress update.
			ingressEvent()

			// Get all the ingresses
			ingressList := testIngress()

			EmptyHealthProbeChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
				Expect((*appGW.Probes)[0]).To(Equal(defaultProbe(appGwIdentifier, n.HTTP)))
			}

			EmptyBackendHTTPSettingsChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
				expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
				httpSettingsName := generateHTTPSettingsName(generateBackendID(ingress, nil, nil, expectedBackend).serviceFullName(), fmt.Sprintf("%d", servicePort), Port(servicePort), ingress.Name)
				httpSettings := &n.ApplicationGatewayBackendHTTPSettings{
					Etag: to.StringPtr("*"),
					Name: &httpSettingsName,
					ID:   to.StringPtr(appGwIdentifier.HTTPSettingsID(httpSettingsName)),
					ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
						Protocol:                       n.HTTP,
						Port:                           to.Int32Ptr(int32(servicePort)),
						Path:                           nil,
						Probe:                          resourceRef(appGwIdentifier.probeID(defaultProbeName(n.HTTP))),
						PickHostNameFromBackendAddress: to.BoolPtr(false),
						CookieBasedAffinity:            n.Disabled,
						RequestTimeout:                 to.Int32Ptr(30),
					},
				}

				// Test the default backend HTTP settings.
				Expect((*appGW.BackendHTTPSettingsCollection)).To(ContainElement(defaultBackendHTTPSettings(appGwIdentifier, n.HTTP)))
				// Test the ingress backend HTTP setting that we installed.
				Expect((*appGW.BackendHTTPSettingsCollection)).To(ContainElement(*httpSettings))
			}

			EmptyBackendAddressPoolChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
				// Test the default backend address pool.
				Expect((*appGW.BackendAddressPools)).To(ContainElement(defaultBackendAddressPool(appGwIdentifier)))
			}

			testAGConfig(ingressList, serviceList, appGwConfigSettings{
				healthProbesCollection: appGWSettingsChecker{
					total:   1,
					checker: EmptyHealthProbeChecker,
				},
				backendHTTPSettingsCollection: appGWSettingsChecker{
					total:   2,
					checker: EmptyBackendHTTPSettingsChecker,
				},
				backendAddressPools: appGWSettingsChecker{
					total:   1,
					checker: EmptyBackendAddressPoolChecker,
				},
				listeners: appGWSettingsChecker{
					total:   1,
					checker: defaultListenersChecker,
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

	Context("Tests Ingress Controller TLS", func() {
		It("Should be able to create Application Gateway Configuration from Ingress with TLS.", func() {
			// Test setup ........................
			// 1. Create secrets object in the Kubernetes secret store.
			ingressSecret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ag-secret",
					Namespace: ingressNS,
				},
				Type: "kubernetes.io/tls",
				Data: make(map[string][]byte),
			}

			key, err := ioutil.ReadFile("../../tests/data/k8s.cert.key")
			Ω(err).ToNot(HaveOccurred(), "Unable to read the cert key: %v", err)
			ingressSecret.Data["tls.key"] = key

			cert, err := ioutil.ReadFile("../../tests/data/k8s.x509.cert")
			Ω(err).ToNot(HaveOccurred(), "Unable to read the cert key: %v", err)
			ingressSecret.Data["tls.crt"] = cert

			// Create a secret in Kubernetes.
			_, err = k8sClient.CoreV1().Secrets(ingressNS).Create(context.TODO(), ingressSecret, metav1.CreateOptions{})
			Ω(err).ToNot(HaveOccurred(), "Unable to create the secret object in K8s: %v", err)

			// 2. Update the ingress TLS spec with a secret from the k8s secret store.
			ingressTLS := v1beta1.IngressTLS{
				SecretName: "test-ag-secret",
			}

			// Currently, when TLS spec is specified for an ingress the expectation is that we will not have any HTTP listeners configured for that ingress.
			// TODO: This statement will not hold true once we introduce the `ssl-redirect` annotation. Will need to rethink this test-case, or introduce a new one.
			// after the introduction of the `ssl-redirect` annotation.
			ingress, err := k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Get(context.TODO(), ingressName, metav1.GetOptions{})
			Ω(err).ToNot(HaveOccurred(), "Unable to create ingress resource due to: %v", err)

			ingress.Spec.TLS = append(ingress.Spec.TLS, ingressTLS)

			// Update the ingress.
			_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Update(context.TODO(), ingress, metav1.UpdateOptions{})
			Ω(err).ToNot(HaveOccurred(), "Unable to update ingress resource due to: %v", err)

			// Start the informers. This will sync the cache with the latest ingress.
			err = ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Ω(err).ToNot(HaveOccurred())

			// Wait for the controller to receive an ingress update.
			ingressEvent()

			// Make sure the ctxt cached the secret.
			secKey := utils.GetResourceKey(ingressNS, "test-ag-secret")

			ctxtSecret := ctxt.GetSecret(secKey)
			Expect(ctxtSecret).To(Equal(ingressSecret))

			pfxCert := ctxt.CertificateSecretStore.GetPfxCertificate(secKey)
			Expect(pfxCert).ShouldNot(BeNil())

			httpsOnlyListenersChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
				// Test the listener.
				secretID := secretIdentifier{
					Namespace: ingressNS,
					Name:      "test-ag-secret",
				}

				frontendPortID := appGwIdentifier.frontendPortID(generateFrontendPortName(443))
				_, httpsListenerName := newTestListenerID(Port(443), []string{domainName}, false)
				sslCert := appGwIdentifier.sslCertificateID(secretID.secretFullName())
				httpsListener := &n.ApplicationGatewayHTTPListener{
					Etag: to.StringPtr("*"),
					Name: &httpsListenerName,
					ID:   to.StringPtr(appGwIdentifier.listenerID(httpsListenerName)),
					ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
						FrontendIPConfiguration: resourceRef("--front-end-ip-id-1--"),
						FrontendPort:            resourceRef(frontendPortID),
						SslCertificate:          resourceRef(sslCert),
						Protocol:                n.HTTPS,

						// RequireServerNameIndication is not used in Application Gateway v2
						RequireServerNameIndication: to.BoolPtr(false),
						HostNames:                   &[]string{domainName},
					},
				}

				Expect(*appGW.HTTPListeners).Should(ConsistOf(*httpsListener))
			}

			// Method to test all the ingress that have been added to the K8s context.
			testTLSIngress := func() []*v1beta1.Ingress {
				// Get all the ingresses
				ingressList := ctxt.ListHTTPIngresses()
				// There should be only one ingress
				Expect(len(ingressList)).To(Equal(1), "Expected only one ingress resource but got: %d", len(ingressList))
				// Make sure it is the ingress we stored.
				Expect(ingressList).To(ContainElement(ingress))

				return ingressList
			}

			// Get all the ingresses
			ingressList := testTLSIngress()

			testAGConfig(ingressList, serviceList, appGwConfigSettings{
				healthProbesCollection: appGWSettingsChecker{
					total:   2,
					checker: defaultHealthProbesChecker,
				},
				backendHTTPSettingsCollection: appGWSettingsChecker{
					total:   2,
					checker: defaultBackendHTTPSettingsChecker,
				},
				backendAddressPools: appGWSettingsChecker{
					total:   2,
					checker: defaultBackendAddressPoolChecker,
				},
				listeners: appGWSettingsChecker{
					total:   1,
					checker: httpsOnlyListenersChecker,
				},
				requestRoutingRules: appGWSettingsChecker{
					total:   1,
					checker: defaultHTTPSRequestRoutingRulesChecker,
				},
				uRLPathMaps: appGWSettingsChecker{
					total:   1,
					checker: defaultHTTPSURLPathMapsChecker,
				},
			})

		})
	})

	Context("Tests Ingress Controller Annotations", func() {
		BeforeEach(func() {
			ingress, err := k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Get(context.TODO(), ingressName, metav1.GetOptions{})
			Ω(err).ToNot(HaveOccurred(), "Unable to get ingress resource due to: %v", err)

			// Set the ingress annotations for this ingress.
			ingress.Annotations[annotations.BackendPathPrefixKey] = "/test"
			ingress.Annotations[annotations.BackendHostNameKey] = "www.backend.com"
			ingress.Annotations[annotations.ConnectionDrainingKey] = "true"
			ingress.Annotations[annotations.ConnectionDrainingTimeoutKey] = "10"
			ingress.Annotations[annotations.CookieBasedAffinityKey] = "true"
			ingress.Annotations[annotations.RequestTimeoutKey] = "10"

			// Update the ingress.
			_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Update(context.TODO(), ingress, metav1.UpdateOptions{})
			Ω(err).ToNot(HaveOccurred(), "Unable to update ingress resource due to: %v", err)

			pod, err := k8sClient.CoreV1().Pods(ingressNS).Get(context.TODO(), serviceName, metav1.GetOptions{})
			Ω(err).ToNot(HaveOccurred(), "Unable to get pod resource due to: %v", err)

			// remove the probe to see the effect of annotations on probe
			pod.Spec.Containers[0].ReadinessProbe = nil
			pod.Spec.Containers[0].LivenessProbe = nil

			// Update the pod.
			_, err = k8sClient.CoreV1().Pods(ingressNS).Update(context.TODO(), pod, metav1.UpdateOptions{})
			Ω(err).ToNot(HaveOccurred(), "Unable to update pod resource due to: %v", err)

			// Start the informers. This will sync the cache with the latest ingress.
			err = ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Ω(err).ToNot(HaveOccurred())

			// Wait for the controller to receive an ingress update.
			ingressEvent()

			// Get all the ingresses
			ingressList := ctxt.ListHTTPIngresses()
			// There should be only one ingress
			Expect(len(ingressList)).To(Equal(1), "Expected only one ingress resource but got: %d", len(ingressList))
			// Make sure it is the ingress we stored.
			Expect(ingressList[0]).To(Equal(ingress))
		})

		It("Should be able to create Application Gateway Configuration from Ingress with all annotations.", func() {

			annotationsHTTPSettingsChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
				expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
				probeID := appGwIdentifier.probeID(generateProbeName(expectedBackend.ServiceName, expectedBackend.ServicePort.String(), ingress))
				httpSettingsName := generateHTTPSettingsName(generateBackendID(ingress, nil, nil, expectedBackend).serviceFullName(), fmt.Sprintf("%d", servicePort), Port(backendPort), ingress.Name)
				httpSettings := &n.ApplicationGatewayBackendHTTPSettings{
					Etag: to.StringPtr("*"),
					Name: &httpSettingsName,
					ID:   to.StringPtr(appGwIdentifier.HTTPSettingsID(httpSettingsName)),
					ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
						Protocol:            n.HTTP,
						Port:                to.Int32Ptr(int32(backendPort)),
						Path:                to.StringPtr("/test"),
						Probe:               resourceRef(probeID),
						HostName:            to.StringPtr("www.backend.com"),
						CookieBasedAffinity: n.Enabled,
						ConnectionDraining: &n.ApplicationGatewayConnectionDraining{
							Enabled:           to.BoolPtr(true),
							DrainTimeoutInSec: to.Int32Ptr(10),
						},
						RequestTimeout:                 to.Int32Ptr(10),
						PickHostNameFromBackendAddress: to.BoolPtr(false),
					},
				}

				backendSettings := *appGW.BackendHTTPSettingsCollection

				defaultHTTPSettings := defaultBackendHTTPSettings(appGwIdentifier, n.HTTP)

				Expect(len(backendSettings)).To(Equal(2))
				// Test the default backend HTTP settings.
				Expect(backendSettings).To(ContainElement(defaultHTTPSettings))
				// Test the ingress backend HTTP setting that we installed.
				Expect(backendSettings).To(ContainElement(*httpSettings))
			}

			annotationHealthProbesChecker := func(appGW *n.ApplicationGatewayPropertiesFormat) {
				expectedBackend := &ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend
				probeName := generateProbeName(expectedBackend.ServiceName, expectedBackend.ServicePort.String(), ingress)
				probe := &n.ApplicationGatewayProbe{
					Name: &probeName,
					ID:   to.StringPtr(appGwIdentifier.probeID(probeName)),
					ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
						Protocol:                            n.HTTP,
						Host:                                to.StringPtr("www.backend.com"),
						Path:                                to.StringPtr("/test"),
						Interval:                            to.Int32Ptr(30),
						UnhealthyThreshold:                  to.Int32Ptr(3),
						Timeout:                             to.Int32Ptr(30),
						Match:                               &n.ApplicationGatewayProbeHealthResponseMatch{},
						PickHostNameFromBackendHTTPSettings: to.BoolPtr(false),
						MinServers:                          to.Int32Ptr(0),
					},
				}

				probes := *appGW.Probes
				Expect(len(probes)).To(Equal(3))

				// Test the default health probe.
				Expect(probes).To(ContainElement(defaultProbe(appGwIdentifier, n.HTTP)))
				// Test the ingress health probe that we installed.
				Expect(probes).To(ContainElement(*probe))
			}
			ingressList := ctxt.ListHTTPIngresses()
			testAGConfig(ingressList, serviceList, appGwConfigSettings{
				healthProbesCollection: appGWSettingsChecker{
					total:   2,
					checker: annotationHealthProbesChecker,
				},
				backendHTTPSettingsCollection: appGWSettingsChecker{
					total:   2,
					checker: annotationsHTTPSettingsChecker,
				},
				backendAddressPools: appGWSettingsChecker{
					total:   2,
					checker: defaultBackendAddressPoolChecker,
				},
				listeners: appGWSettingsChecker{
					total:   1,
					checker: defaultListenersChecker,
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

	Context("Tests Application Gateway Generate HTTP Settings Name", func() {
		It("Should be create an Application Gateway Backend Pool Name With Less than 80 Characters", func() {
			// Start the informers. This will sync the cache with the latest ingress.
			err := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Ω(err).ToNot(HaveOccurred())

			// Wait for the controller to receive an ingress update.
			ingressEvent()

			serviceName := "test-cm-acme-http-solver-j7sxh"
			servicePort := "8089"
			backendPortNo := Port(8089)
			ingress := "cm-acme-http-solver-t8rnf"

			httpSettingsName := generateHTTPSettingsName(serviceName, servicePort, Port(backendPortNo), ingress)
			Ω(len(httpSettingsName)).Should(BeNumerically("<=", 80), "Expected App Gateway Backend Pool with 80 Character but got one with: %d", len(httpSettingsName))
		})
	})
})
