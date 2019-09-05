// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Process ingress rules and parse frontend listener configs", func() {
	envVariables := environment.GetFakeEnv()

	listener80 := listenerIdentifier{
		FrontendPort: Port(80),
		HostName:     tests.Host,
	}

	listenerAzConfigNoSSL := listenerAzConfig{
		Protocol: "Http",
		Secret: secretIdentifier{
			Namespace: "",
			Name:      "",
		},
		SslRedirectConfigurationName: "",
	}

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
			Expect(azConfigMapKeys).To(ContainElement(listener80))
			actualVal := httpListenersAzureConfigMap[listener80]
			Expect(actualVal).To(Equal(listenerAzConfigNoSSL))
		})
	})
	Context("two ingresses with multiple ports", func() {
		certs := newCertsFixture()
		cb := newConfigBuilderFixture(&certs)

		ing1 := tests.NewIngressFixture()
		ing2 := tests.NewIngressFixture()
		ingressList := []*v1beta1.Ingress{
			ing1,
			ing2,
		}

		cbCtx := &ConfigBuilderContext{
			IngressList:  ingressList,
			EnvVariables: envVariables,
		}

		// !! Action !!
		cb.appGw.FrontendPorts = cb.getFrontendPorts(cbCtx)
		listeners := cb.getListeners(cbCtx)

		It("should have correct number of listeners", func() {
			Expect(len(*listeners)).To(Equal(2))
		})

		It("should have correct values for listeners", func() {
			// Get the HTTPS listener for this test
			var listener n.ApplicationGatewayHTTPListener
			for _, listener = range *listeners {
				if listener.Protocol == "Https" && *listener.HostName == tests.Host {
					break
				}
			}

			Expect(*listener.HostName).To(Equal(tests.Host))
			Expect(*listener.FrontendPort.ID).To(Equal(cb.appGwIdentifier.frontendPortID(generateFrontendPortName(443))))

			expectedProtocol := n.ApplicationGatewayProtocol("Https")
			Expect(listener.Protocol).To(Equal(expectedProtocol))

			Expect(*listener.FrontendIPConfiguration.ID).To(Equal(tests.PublicIPID))
		})
	})
	Context("create a new App Gateway HTTP Listener", func() {
		It("should create a correct App Gwy listener", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			ing1 := tests.NewIngressFixture()
			ing2 := tests.NewIngressFixture()
			ingressList := []*v1beta1.Ingress{
				ing1,
				ing2,
			}

			cbCtx := &ConfigBuilderContext{
				IngressList:  ingressList,
				EnvVariables: envVariables,
			}

			cb.appGw.FrontendPorts = cb.getFrontendPorts(cbCtx)
			listener := cb.newListener(listener80, n.ApplicationGatewayProtocol("Https"))
			expectedName := agPrefix + "fl-bye.com-80-pub"

			expected := n.ApplicationGatewayHTTPListener{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(expectedName),
				ID:   to.StringPtr(cb.appGwIdentifier.listenerID(expectedName)),
				ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
					// TODO: expose this to external configuration
					FrontendIPConfiguration: resourceRef(tests.PublicIPID),
					FrontendPort:            resourceRef(cb.appGwIdentifier.frontendPortID(generateFrontendPortName(80))),
					Protocol:                n.ApplicationGatewayProtocol("Https"),
					HostName:                to.StringPtr(tests.Host),
				},
			}

			Expect(listener).To(Equal(expected))
		})
	})
	Context("create a new App Gateway HTTP Listener with Private Ip when environment USE_PRIVATE_IP is true", func() {
		const (
			expectedEnvVarValue = "true"
		)
		envVariablesNew := environment.GetFakeEnv()
		envVariablesNew.UsePrivateIP = expectedEnvVarValue
		It("should have usePrivateIP true", func() {
			Expect(envVariablesNew.UsePrivateIP).To(Equal(expectedEnvVarValue))
		})
		It("should create a App Gwy listener with private IP", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			ing1 := tests.NewIngressFixture()
			ing2 := tests.NewIngressFixture()
			cbCtx := &ConfigBuilderContext{
				IngressList: []*v1beta1.Ingress{
					ing1,
					ing2,
				},
				EnvVariables: envVariablesNew,
			}
			cb.appGw.FrontendPorts = cb.getFrontendPorts(cbCtx)
			for listenerID, listenerAzConfig := range cb.getListenerConfigs(cbCtx) {
				listener := cb.newListener(listenerID, listenerAzConfig.Protocol)
				Expect(*listener.FrontendIPConfiguration.ID).To(Equal(tests.PrivateIPID))
			}
		})
	})

	Context("create a new App Gateway HTTP Listener with Private Ip when usePrivateIP annotation is present", func() {
		listener80Private := listenerIdentifier{
			FrontendPort: Port(80),
			HostName:     tests.Host,
			UsePrivateIP: true,
		}
		It("should have usePrivateIP true", func() {
			Expect(listener80Private.UsePrivateIP).To(Equal(true))
		})
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
			cb.appGw.FrontendPorts = cb.getFrontendPorts(cbCtx)
			listener := cb.newListener(listener80Private, n.ApplicationGatewayProtocol("Https"))
			expectedName := agPrefix + "fl-bye.com-80-priv"

			expected := n.ApplicationGatewayHTTPListener{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(expectedName),
				ID:   to.StringPtr(cb.appGwIdentifier.listenerID(expectedName)),
				ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
					FrontendIPConfiguration: resourceRef(tests.PrivateIPID),
					FrontendPort:            resourceRef(cb.appGwIdentifier.frontendPortID(generateFrontendPortName(80))),
					Protocol:                n.ApplicationGatewayProtocol("Https"),
					HostName:                to.StringPtr(tests.Host),
				},
			}

			Expect(listener).To(Equal(expected))
		})
	})
})
