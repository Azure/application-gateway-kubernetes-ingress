// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"math/rand"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Process ingress rules and parse frontend listener configs", func() {

	var envVariables environment.EnvVariables
	var listenerID80 listenerIdentifier
	var listenerID80Priv listenerIdentifier
	var listenerID443 listenerIdentifier
	var listenerAzConfigNoSSL listenerAzConfig
	var listenerAzConfigWithSSL listenerAzConfig
	var expectedPort80 n.ApplicationGatewayFrontendPort
	var expectedPort443 n.ApplicationGatewayFrontendPort
	var expectedListener80 n.ApplicationGatewayHTTPListener
	var expectedListener80Priv n.ApplicationGatewayHTTPListener
	var expectedListener443 n.ApplicationGatewayHTTPListener
	var expectedListener443Priv n.ApplicationGatewayHTTPListener

	resPref := "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/"

	BeforeEach(func() {
		envVariables = environment.GetFakeEnv()

		listenerID80 = listenerIdentifier{
			FrontendPort: Port(80),
			HostName:     tests.Host,
		}

		listenerID80Priv = listenerIdentifier{
			FrontendPort: Port(80),
			HostName:     tests.Host,
			UsePrivateIP: true,
		}

		listenerID443 = listenerIdentifier{
			FrontendPort: Port(443),
			HostName:     tests.Host,
		}

		listenerAzConfigNoSSL = listenerAzConfig{
			Protocol: "Http",
			Secret: secretIdentifier{
				Namespace: "",
				Name:      "",
			},
			SslRedirectConfigurationName: "",
		}

		listenerAzConfigWithSSL = listenerAzConfig{
			Protocol: "Https",
			Secret: secretIdentifier{
				Namespace: "--namespace--",
				Name:      "--the-name-of-the-secret--",
			},
			SslRedirectConfigurationName: "sslr-fl-bye.com-443",
		}

		expectedPort80 = n.ApplicationGatewayFrontendPort{
			ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
				Port: to.Int32Ptr(80),
			},
			Name: to.StringPtr("fp-80"),
			Etag: to.StringPtr("*"),
			ID:   to.StringPtr(resPref + "frontendPorts/fp-80"),
		}

		expectedPort443 = n.ApplicationGatewayFrontendPort{
			ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
				Port: to.Int32Ptr(443),
			},
			Name: to.StringPtr("fp-443"),
			Etag: to.StringPtr("*"),
			ID:   to.StringPtr(resPref + "frontendPorts/fp-443"),
		}

		expectedListener80 = n.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr("fl-bye.com-80"),
			ID:   to.StringPtr(resPref + "httpListeners/fl-bye.com-80"),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration: resourceRef(tests.PublicIPID),
				FrontendPort:            resourceRef(resPref + "frontendPorts/fp-80"),
				Protocol:                n.ApplicationGatewayProtocol("Http"),
				HostName:                to.StringPtr(tests.Host),
			},
		}

		expectedListener80Priv = n.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr("fl-bye.com-80-privateip"),
			ID:   to.StringPtr(resPref + "httpListeners/fl-bye.com-80-privateip"),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration: resourceRef(tests.PrivateIPID),
				FrontendPort:            resourceRef(resPref + "frontendPorts/fp-80"),
				Protocol:                n.ApplicationGatewayProtocol("Http"),
				HostName:                to.StringPtr(tests.Host),
			},
		}

		expectedListener443 = n.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr("fl-bye.com-443"),
			ID:   to.StringPtr(resPref + "httpListeners/fl-bye.com-443"),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration: resourceRef(tests.PublicIPID),
				FrontendPort:            resourceRef(resPref + "frontendPorts/fp-443"),
				Protocol:                n.ApplicationGatewayProtocol("Https"),
				HostName:                to.StringPtr(tests.Host),
				SslCertificate:          resourceRef(resPref + "sslCertificates/--namespace-----the-name-of-the-secret--"),
			},
		}

		expectedListener443Priv = n.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr("fl-bye.com-443-privateip"),
			ID:   to.StringPtr(resPref + "httpListeners/fl-bye.com-443-privateip"),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration: resourceRef(tests.PrivateIPID),
				FrontendPort:            resourceRef(resPref + "frontendPorts/fp-443"),
				Protocol:                n.ApplicationGatewayProtocol("Https"),
				HostName:                to.StringPtr(tests.Host),
				SslCertificate:          resourceRef(resPref + "sslCertificates/--namespace-----the-name-of-the-secret--"),
			},
		}
	})

	Context("ingress rules without certificates", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)
		ingress := tests.NewIngressFixture()
		cbCtx := &ConfigBuilderContext{
			IngressList: []*v1beta1.Ingress{ingress},
		}

		httpListenersAzureConfigMap := cb.getListenerConfigs(cbCtx)

		It("should construct the App Gateway listeners correctly without SSL", func() {
			azConfigMapKeys := getMapKeys(&httpListenersAzureConfigMap)
			Expect(len(azConfigMapKeys)).To(Equal(2))
			Expect(azConfigMapKeys).To(ContainElement(listenerID80))
			Expect(azConfigMapKeys).To(ContainElement(listenerID443))
			Expect(httpListenersAzureConfigMap[listenerID80]).To(Equal(listenerAzConfigNoSSL))
			Expect(httpListenersAzureConfigMap[listenerID443]).To(Equal(listenerAzConfigWithSSL))
		})
	})

	Context("Use newListener() to create a new App Gateway HTTP Listener", func() {
		It("should create a correct App Gwy listener", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					tests.NewIngressFixture(),
					tests.NewIngressFixture(),
				},
				EnvVariables: envVariables,
			}

			listener, port, err := cb.newListener(cbCtx, listenerID80, n.ApplicationGatewayProtocol("Https"))
			Expect(err).ToNot(HaveOccurred())
			expectedListener80.ApplicationGatewayHTTPListenerPropertiesFormat.Protocol = n.ApplicationGatewayProtocol("Https")

			Expect(*listener).To(Equal(expectedListener80))
			Expect(*port).To(Equal(expectedPort80))

		})
	})

	Context("Use getListenerConfigs() to create a new App Gateway HTTP Listener with Private Ip when environment USE_PRIVATE_IP is true", func() {
		It("should create a App Gwy listener with private IP", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					tests.NewIngressFixture(),
					tests.NewIngressFixture(),
				},
				EnvVariables: envVariables,
			}
			cbCtx.EnvVariables.UsePrivateIP = "true"

			listeners, ports := cb.getListeners(cbCtx)
			Expect(len(*listeners)).To(Equal(2))
			Expect(len(*ports)).To(Equal(2))

			listenerConfigs := cb.getListenerConfigs(cbCtx)

			{
				listenerID := listenerIdentifier{80, "bye.com", true}
				listenerAzConfig, exists := listenerConfigs[listenerID]
				Expect(exists).To(BeTrue())
				listener, port, err := cb.newListener(cbCtx, listenerID, listenerAzConfig.Protocol)
				Expect(err).ToNot(HaveOccurred())
				Expect(*listener.FrontendIPConfiguration.ID).To(Equal(tests.PrivateIPID))
				Expect(*port).To(Equal(expectedPort80))
			}

			{
				listenerID := listenerIdentifier{443, "bye.com", true}
				listenerAzConfig, exists := listenerConfigs[listenerID]
				Expect(exists).To(BeTrue())
				listener, port, err := cb.newListener(cbCtx, listenerID, listenerAzConfig.Protocol)
				Expect(err).ToNot(HaveOccurred())
				Expect(*listener.FrontendIPConfiguration.ID).To(Equal(tests.PrivateIPID))
				Expect(*port).To(Equal(expectedPort443))
			}
		})
	})

	Context("create a new App Gateway HTTP Listener with Private IP when usePrivateIP annotation is present", func() {
		It("should create a App Gwy listener with private IP", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					tests.NewIngressFixture(),
				},
				EnvVariables: envVariables,
			}

			listeners, ports := cb.getListeners(cbCtx)
			Expect(len(*listeners)).To(Equal(2))
			Expect(len(*ports)).To(Equal(2))

			listener, port, err := cb.newListener(cbCtx, listenerID80Priv, n.ApplicationGatewayProtocol("Http"))
			Expect(err).ToNot(HaveOccurred())
			Expect(*listener).To(Equal(expectedListener80Priv))
			Expect(*port).To(Equal(expectedPort80))
		})
	})

	Context("create a new App Gateway HTTP Listener with Private IP when usePrivateIP annotation is present", func() {
		It("should create a App Gwy listener with private IP", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			envVariablesCopy := envVariables
			envVariablesCopy.UsePrivateIP = "true"
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					tests.NewIngressFixture(),
					tests.NewIngressFixture(),
				},
				EnvVariables: envVariablesCopy,
			}

			cbCtx.IngressList[0].Annotations[annotations.UsePrivateIPKey] = "true"

			listeners, ports := cb.getListeners(cbCtx)
			Expect(len(*listeners)).To(Equal(2))
			Expect(len(*ports)).To(Equal(2))

			Expect(*listeners).To(ContainElement(expectedListener80Priv))
			Expect(*listeners).To(ContainElement(expectedListener443Priv))

			Expect(*ports).To(ContainElement(expectedPort80))
			Expect(*ports).To(ContainElement(expectedPort443))
		})
	})

	Context("many listeners, same port", func() {
		It("should create only one listener", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					tests.NewIngressFixture(),
				},
				EnvVariables: envVariables,
			}

			// Add a bunch of Ingress Resources
			rand.Seed(time.Now().UnixNano())
			for range make([]interface{}, rand.Intn(99)) {
				cbCtx.IngressList = append(cbCtx.IngressList, tests.NewIngressFixture())
			}

			listeners, ports := cb.getListeners(cbCtx)
			Expect(len(*listeners)).To(Equal(2))
			Expect(len(*ports)).To(Equal(2))

			Expect(*listeners).To(ContainElement(expectedListener80))
			Expect(*listeners).To(ContainElement(expectedListener443))

			Expect(*ports).To(ContainElement(expectedPort80))
			Expect(*ports).To(ContainElement(expectedPort443))
		})
	})

	Context("create a new App Gateway HTTP Listener for V1 gateway", func() {
		ing1 := tests.NewIngressFixture()
		ing2 := tests.NewIngressFixture()
		ingressList := []*v1beta1.Ingress{
			ing1,
			ing2,
		}

		listenerID80WithoutHostname := listenerIdentifier{
			FrontendPort: Port(80),
			HostName:     "",
		}

		cbCtx := &ConfigBuilderContext{
			IngressList:  ingressList,
			EnvVariables: envVariables,
		}

		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)

		// V1 gateway
		cb.appGw.Sku = &n.ApplicationGatewaySku{
			Name:     n.StandardLarge,
			Tier:     n.ApplicationGatewayTierStandard,
			Capacity: to.Int32Ptr(3),
		}

		It("should create listener with RequireServerNameIndication when (https, hostname) listener", func() {
			listener, _, _ := cb.newListener(cbCtx, listenerID80, n.ApplicationGatewayProtocol("Https"))
			Expect(*listener.RequireServerNameIndication).To(BeTrue())
		})

		It("should not create listener with RequireServerNameIndication when (https, no hostname) listener", func() {
			listener, _, _ := cb.newListener(cbCtx, listenerID80WithoutHostname, n.ApplicationGatewayProtocol("Https"))
			Expect(len(*listener.HostName)).To(Equal(0))
			Expect(listener.RequireServerNameIndication).To(BeNil())
		})

		It("should not create listener with RequireServerNameIndication when (http, hostname) listener", func() {
			listener, _, _ := cb.newListener(cbCtx, listenerID80, n.ApplicationGatewayProtocol("Http"))
			Expect(listener.RequireServerNameIndication).To(BeNil())
		})

		It("should not create listener with RequireServerNameIndication when (http, no hostname) listener", func() {
			listener, _, _ := cb.newListener(cbCtx, listenerID80WithoutHostname, n.ApplicationGatewayProtocol("Http"))
			Expect(len(*listener.HostName)).To(Equal(0))
			Expect(listener.RequireServerNameIndication).To(BeNil())
		})
	})
})
