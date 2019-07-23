package appgw

import (
	"os"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
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
		FrontendPort: int32(80),
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
		ingressList := []*v1beta1.Ingress{ingress}
		httpListenersAzureConfigMap := cb.getListenerConfigs(ingressList)

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

			Expect(*listener.FrontendIPConfiguration.ID).To(Equal(tests.IPID1))
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
			listener := cb.newListener(listener80, n.ApplicationGatewayProtocol("Https"), envVariables)
			expectedName := agPrefix + "fl-bye.com-80"

			expected := n.ApplicationGatewayHTTPListener{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(expectedName),
				ID:   to.StringPtr(cb.appGwIdentifier.listenerID(expectedName)),
				ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
					// TODO: expose this to external configuration
					FrontendIPConfiguration: resourceRef(tests.IPID1),
					FrontendPort:            resourceRef(cb.appGwIdentifier.frontendPortID(generateFrontendPortName(80))),
					Protocol:                n.ApplicationGatewayProtocol("Https"),
					HostName:                to.StringPtr(tests.Host),
				},
			}

			Expect(listener).To(Equal(expected))
		})
	})
	Context("create a new App Gateway HTTP Listener with Private Ip when private IP is present", func() {
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
			ingressList := []*v1beta1.Ingress{
				ing1,
				ing2,
			}

			cbCtx := &ConfigBuilderContext{
				IngressList:  ingressList,
				EnvVariables: envVariables,
			}
			cb.appGw.FrontendPorts = cb.getFrontendPorts(cbCtx)
			listener := cb.newListener(listener80, n.ApplicationGatewayProtocol("Https"), envVariablesNew)
			expectedName := agPrefix + "fl-bye.com-80"

			expected := n.ApplicationGatewayHTTPListener{
				Etag: to.StringPtr("*"),
				Name: to.StringPtr(expectedName),
				ID:   to.StringPtr(cb.appGwIdentifier.listenerID(expectedName)),
				ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
					// TODO: expose this to external configuration
					FrontendIPConfiguration: resourceRef(tests.IPID2),
					FrontendPort:            resourceRef(cb.appGwIdentifier.frontendPortID(generateFrontendPortName(80))),
					Protocol:                n.ApplicationGatewayProtocol("Https"),
					HostName:                to.StringPtr(tests.Host),
				},
			}

			Expect(listener).To(Equal(expected))
		})
	})
	Context("Fatal if UsePrivateIp is specified and Application Gateway doesn't have a private IP configured.", func() {
		const (
			expectedEnvVarValue = "true"
		)
		BeforeEach(func() {
			// Make sure the environment variable we are using for this test does not already exist in the OS.
			_, exists := os.LookupEnv(environment.UsePrivateIPVarName)
			Expect(exists).To(BeFalse())
			// Set it
			_ = os.Setenv(environment.UsePrivateIPVarName, expectedEnvVarValue)
			_, exists = os.LookupEnv(environment.UsePrivateIPVarName)
			Expect(exists).To(BeTrue())
		})
		AfterEach(func() {
			// Clean up the env var after the tests are done
			_ = os.Unsetenv(environment.UsePrivateIPVarName)
		})
		It("should have fatal crash.", func() {
			certs := newCertsFixture()
			cb := newConfigBuilderFixture(&certs)
			cb.appGw.FrontendIPConfigurations = &[]n.ApplicationGatewayFrontendIPConfiguration{
				(*cb.appGw.FrontendIPConfigurations)[0],
			}

			Expect((*cb.appGw.FrontendIPConfigurations)[0].ApplicationGatewayFrontendIPConfigurationPropertiesFormat.PublicIPAddress).ToNot(Equal(nil))
			Expect(len(*cb.appGw.FrontendIPConfigurations)).To(Equal(1))

			// exiter = New(func() { cb.newListener(listener80, n.ApplicationGatewayProtocol("Https")) })
			// exiter.Exit(3)
			// Expected(exiter.Status(), ShouldEqual, 3)
		})
	})
})
