// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"math/rand"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	networking "k8s.io/api/networking/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("MutateAppGateway ingress rules and parse frontend listener configs", func() {

	var envVariables environment.EnvVariables
	var listenerID80 listenerIdentifier
	var listenerID80Priv listenerIdentifier
	var listenerID443 listenerIdentifier
	var listenerID80ExtendedHost listenerIdentifier
	var listenerAzConfigNoSSL listenerAzConfig
	var listenerAzConfigWithSSL listenerAzConfig
	var expectedPort80 n.ApplicationGatewayFrontendPort
	var expectedPort443 n.ApplicationGatewayFrontendPort
	var expectedListener80 n.ApplicationGatewayHTTPListener
	var expectedListener80Priv n.ApplicationGatewayHTTPListener
	var expectedListener443 n.ApplicationGatewayHTTPListener
	var expectedListener443Priv n.ApplicationGatewayHTTPListener
	var expectedListener80MultiHostNames n.ApplicationGatewayHTTPListener
	var listenerID80Name string
	var listenerID80PrivName string
	var listenerID443Name string
	var listenerID443PrivName string
	var listenerID80ExtendedHostName string

	resPref := "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/"

	BeforeEach(func() {
		envVariables = environment.GetFakeEnv()

		listenerID80, listenerID80Name = newTestListenerID(Port(80), []string{tests.Host}, false)

		listenerID80ExtendedHost, listenerID80ExtendedHostName = newTestListenerID(Port(80), []string{"test.com", "t*.com"}, false)

		listenerID80Priv, listenerID80PrivName = newTestListenerID(Port(80), []string{tests.Host}, true)

		listenerID443, listenerID443Name = newTestListenerID(Port(443), []string{tests.Host}, false)

		_, listenerID443PrivName = newTestListenerID(Port(443), []string{tests.Host}, true)

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
			SslRedirectConfigurationName: generateSSLRedirectConfigurationName(listenerID443),
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
			Name: to.StringPtr(listenerID80Name),
			ID:   to.StringPtr(resPref + "httpListeners/" + listenerID80Name),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration:     resourceRef(tests.PublicIPID),
				FrontendPort:                resourceRef(resPref + "frontendPorts/fp-80"),
				Protocol:                    n.ApplicationGatewayProtocol("Http"),
				RequireServerNameIndication: to.BoolPtr(false),
				HostNames:                   &[]string{tests.Host},
			},
		}

		expectedListener80MultiHostNames = n.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(listenerID80ExtendedHostName),
			ID:   to.StringPtr(resPref + "httpListeners/" + listenerID80ExtendedHostName),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration:     resourceRef(tests.PublicIPID),
				FrontendPort:                resourceRef(resPref + "frontendPorts/fp-80"),
				Protocol:                    n.ApplicationGatewayProtocol("Http"),
				HostNames:                   to.StringSlicePtr([]string{"test.com", "t*.com"}),
				RequireServerNameIndication: to.BoolPtr(false),
			},
		}

		expectedListener80Priv = n.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(listenerID80PrivName),
			ID:   to.StringPtr(resPref + "httpListeners/" + listenerID80PrivName),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration:     resourceRef(tests.PrivateIPID),
				FrontendPort:                resourceRef(resPref + "frontendPorts/fp-80"),
				Protocol:                    n.ApplicationGatewayProtocol("Http"),
				RequireServerNameIndication: to.BoolPtr(false),
				HostNames:                   &[]string{tests.Host},
			},
		}

		expectedListener443 = n.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(listenerID443Name),
			ID:   to.StringPtr(resPref + "httpListeners/" + listenerID443Name),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration:     resourceRef(tests.PublicIPID),
				FrontendPort:                resourceRef(resPref + "frontendPorts/fp-443"),
				Protocol:                    n.ApplicationGatewayProtocol("Https"),
				SslCertificate:              resourceRef(resPref + "sslCertificates/cert---namespace-----the-name-of-the-secret--"),
				RequireServerNameIndication: to.BoolPtr(false),
				HostNames:                   &[]string{tests.Host},
			},
		}

		expectedListener443Priv = n.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(listenerID443PrivName),
			ID:   to.StringPtr(resPref + "httpListeners/" + listenerID443PrivName),
			ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
				FrontendIPConfiguration:     resourceRef(tests.PrivateIPID),
				FrontendPort:                resourceRef(resPref + "frontendPorts/fp-443"),
				Protocol:                    n.ApplicationGatewayProtocol("Https"),
				SslCertificate:              resourceRef(resPref + "sslCertificates/cert---namespace-----the-name-of-the-secret--"),
				RequireServerNameIndication: to.BoolPtr(false),
				HostNames:                   &[]string{tests.Host},
			},
		}
	})

	Context("ingress rules without certificates", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)
		ingress := tests.NewIngressFixture()
		cbCtx := &ConfigBuilderContext{
			IngressList:           []*networking.Ingress{ingress},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
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
				IngressList: []*networking.Ingress{
					tests.NewIngressFixture(),
					tests.NewIngressFixture(),
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			ports := make(map[Port]n.ApplicationGatewayFrontendPort)
			listener, port, err := cb.newListener(cbCtx, listenerID80, n.ApplicationGatewayProtocol("Https"), ports)
			Expect(err).ToNot(HaveOccurred())
			expectedListener80.ApplicationGatewayHTTPListenerPropertiesFormat.Protocol = n.ApplicationGatewayProtocol("Https")

			Expect(*listener).To(Equal(expectedListener80))
			Expect(*port).To(Equal(expectedPort80))
		})

		It("should create a correct App Gwy listener when ingress has extended HostNames", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					tests.NewIngressFixture(),
					tests.NewIngressFixture(),
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			ports := make(map[Port]n.ApplicationGatewayFrontendPort)
			listener, port, err := cb.newListener(cbCtx, listenerID80ExtendedHost, n.ApplicationGatewayProtocol("Http"), ports)
			Expect(err).ToNot(HaveOccurred())

			Expect(*listener).To(Equal(expectedListener80MultiHostNames))
			Expect(*port).To(Equal(expectedPort80))
		})
	})

	Context("Use getListenerConfigs() to create a new App Gateway HTTP Listener with Private Ip when environment USE_PRIVATE_IP is true", func() {
		It("should create a App Gwy listener with private IP", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					tests.NewIngressFixture(),
					tests.NewIngressFixture(),
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}
			cbCtx.EnvVariables.UsePrivateIP = true

			listeners, ports := cb.getListeners(cbCtx)
			Expect(len(*listeners)).To(Equal(2))
			Expect(len(*ports)).To(Equal(2))

			listenerConfigs := cb.getListenerConfigs(cbCtx)

			{
				listenerID, _ := newTestListenerID(Port(80), []string{tests.Host}, true)
				listenerAzConfig, exists := listenerConfigs[listenerID]
				Expect(exists).To(BeTrue())
				ports := make(map[Port]n.ApplicationGatewayFrontendPort)
				listener, port, err := cb.newListener(cbCtx, listenerID, listenerAzConfig.Protocol, ports)
				Expect(err).ToNot(HaveOccurred())
				Expect(*listener.FrontendIPConfiguration.ID).To(Equal(tests.PrivateIPID))
				Expect(*port).To(Equal(expectedPort80))
			}

			{
				listenerID, _ := newTestListenerID(Port(443), []string{tests.Host}, true)
				listenerAzConfig, exists := listenerConfigs[listenerID]
				Expect(exists).To(BeTrue())
				ports := make(map[Port]n.ApplicationGatewayFrontendPort)
				listener, port, err := cb.newListener(cbCtx, listenerID, listenerAzConfig.Protocol, ports)
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
				IngressList: []*networking.Ingress{
					tests.NewIngressFixture(),
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			listeners, ports := cb.getListeners(cbCtx)
			Expect(len(*listeners)).To(Equal(2))
			Expect(len(*ports)).To(Equal(2))
			portsByNumber := make(map[Port]n.ApplicationGatewayFrontendPort)
			listener, port, err := cb.newListener(cbCtx, listenerID80Priv, n.ApplicationGatewayProtocol("Http"), portsByNumber)
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
			envVariablesCopy.UsePrivateIP = true
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					tests.NewIngressFixture(),
					tests.NewIngressFixture(),
				},
				EnvVariables:          envVariablesCopy,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
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
				IngressList: []*networking.Ingress{
					tests.NewIngressFixture(),
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
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

	Context("create a new App Gateway with annotated certificate", func() {
		It("should create listener with certificate", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			ing := tests.NewIngressFixture()
			ing.Annotations[annotations.AppGwSslCertificate] = "appgw-installed-cert"
			ing.Spec.TLS = nil
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					ing,
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			listeners, _ := cb.getListeners(cbCtx)

			expectedListener443.SslCertificate = resourceRef(resPref + "sslCertificates/appgw-installed-cert")
			Expect(*listeners).To(ContainElement(expectedListener443))
		})
	})

	Context("create a new App Gateway with annotated waf policy", func() {
		It("should create listener without waf policy attached when TLS is enabled by ingress TLS Spec", func() {
			wafPolicyID := "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/testwafpolicy1"
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			ing := tests.NewIngressFixture()
			ing.Annotations[annotations.FirewallPolicy] = wafPolicyID
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					ing,
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			listeners, _ := cb.getListeners(cbCtx)
			expectedListener80.FirewallPolicy = resourceRef(wafPolicyID)
			expectedListener443.FirewallPolicy = resourceRef(wafPolicyID)
			Expect(*listeners).NotTo(ContainElement(expectedListener80))
			Expect(*listeners).NotTo(ContainElement(expectedListener443))
		})

		It("should create listener without waf policy attached when TLS is enabled by annotation", func() {
			wafPolicyID := "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/testwafpolicy2"
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			ing := tests.NewIngressFixture()
			ing.Spec.TLS = nil
			ing.Annotations[annotations.AppGwSslCertificate] = "appgw-installed-cert"
			ing.Annotations[annotations.FirewallPolicy] = wafPolicyID
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					ing,
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			listeners, _ := cb.getListeners(cbCtx)
			expectedListener80.FirewallPolicy = resourceRef(wafPolicyID)
			expectedListener443.SslCertificate = resourceRef(resPref + "sslCertificates/appgw-installed-cert")
			expectedListener443.FirewallPolicy = resourceRef(wafPolicyID)
			Expect(*listeners).NotTo(ContainElement(expectedListener80))
			Expect(*listeners).NotTo(ContainElement(expectedListener443))
		})

		It("should create listener without waf policy attached when TLS is disabled", func() {
			wafPolicyID := "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/testwafpolicy2"
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			ing := tests.NewIngressFixture()
			ing.Spec.TLS = nil
			ing.Annotations[annotations.FirewallPolicy] = wafPolicyID
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					ing,
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			listeners, _ := cb.getListeners(cbCtx)
			expectedListener80.FirewallPolicy = resourceRef(wafPolicyID)
			Expect(*listeners).NotTo(ContainElement(expectedListener80))
		})

		It("should create listener with waf policy attached when TLS is enabled by ingress TLS spec", func() {
			wafPolicyID := "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/testwafpolicy3"
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			ing := tests.NewIngressFixtureSingleSlashPath()
			ing.Annotations[annotations.FirewallPolicy] = wafPolicyID
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					ing,
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			listeners, _ := cb.getListeners(cbCtx)
			expectedListener80.FirewallPolicy = resourceRef(wafPolicyID)
			Expect(*listeners).To(ContainElement(expectedListener80))
			expectedListener443.FirewallPolicy = resourceRef(wafPolicyID)
			Expect(*listeners).To(ContainElement(expectedListener443))
		})

		It("should create listener with waf policy attached when TLS is enabled by annotation", func() {
			wafPolicyID := "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/testwafpolicy3"
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			ing := tests.NewIngressFixtureSingleSlashPath()
			ing.Spec.TLS = nil
			ing.Annotations[annotations.AppGwSslCertificate] = "appgw-installed-cert"
			ing.Annotations[annotations.FirewallPolicy] = wafPolicyID
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					ing,
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			listeners, _ := cb.getListeners(cbCtx)

			expectedListener80.FirewallPolicy = resourceRef(wafPolicyID)
			Expect(*listeners).To(ContainElement(expectedListener80))
			expectedListener443.SslCertificate = resourceRef(resPref + "sslCertificates/appgw-installed-cert")
			expectedListener443.FirewallPolicy = resourceRef(wafPolicyID)
			Expect(*listeners).To(ContainElement(expectedListener443))
		})

		It("should create listener with waf policy attached when TLS is disabled", func() {
			wafPolicyID := "/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/testwafpolicy3"
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			// ingress with single slash as rule path
			ing := tests.NewIngressFixtureSingleSlashPath()
			ing.Spec.TLS = nil
			ing.Annotations[annotations.FirewallPolicy] = wafPolicyID
			cbCtx := &ConfigBuilderContext{
				IngressList: []*networking.Ingress{
					ing,
				},
				EnvVariables:          envVariables,
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			listeners, _ := cb.getListeners(cbCtx)

			expectedListener80.FirewallPolicy = resourceRef(wafPolicyID)
			Expect(*listeners).To(ContainElement(expectedListener80))
		})
	})
})
