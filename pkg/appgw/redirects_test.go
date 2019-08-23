// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

var _ = Describe("Test SSL Redirect Annotations", func() {

	listenerID1 := listenerIdentifier{
		FrontendPort: 80,
		HostName:     "bye.com",
	}
	listenerID2 := listenerIdentifier{
		FrontendPort: 443,
		HostName:     "bye.com",
	}

	expectedListenerConfigs := map[listenerIdentifier]listenerAzConfig{
		listenerID1: {
			Protocol: "Http",
		},
		listenerID2: {
			Protocol: "Https",
			Secret: secretIdentifier{
				Namespace: tests.Namespace,
				Name:      "--the-name-of-the-secret--",
			},
			SslRedirectConfigurationName: "sslr-fl-bye.com-443",
		},
	}

	Context("Test RequestRoutingRules with TLS and with SSL Redirect Annotation", func() {
		cb := newConfigBuilderFixture(nil)
		ingress := tests.NewIngressFixture()
		ingressList := []*v1beta1.Ingress{ingress}
		cbCtx := ConfigBuilderContext{
			IngressList: ingressList,
		}
		actualRedirects := cb.getRedirectConfigurations(&cbCtx)
		expectedRedirect := n.ApplicationGatewayRedirectConfiguration{
			ApplicationGatewayRedirectConfigurationPropertiesFormat: &n.ApplicationGatewayRedirectConfigurationPropertiesFormat{
				RedirectType: "Permanent",
				TargetListener: &n.SubResource{
					ID: to.StringPtr("/subscriptions/--subscription--" +
						"/resourceGroups/--resource-group--" +
						"/providers/Microsoft.Network" +
						"/applicationGateways/--app-gw-name--" +
						"/httpListeners/fl-bye.com-443"),
				},
				TargetURL:           nil,
				IncludePath:         to.BoolPtr(true),
				IncludeQueryString:  to.BoolPtr(true),
				RequestRoutingRules: nil,
				URLPathMaps:         nil,
				PathRules:           nil,
			},
			Name: to.StringPtr("sslr-fl-bye.com-443"),
			Etag: to.StringPtr("*"),
			Type: nil,
			ID:   to.StringPtr(cb.appGwIdentifier.redirectConfigurationID("sslr-fl-bye.com-443")),
		}

		actualListeners := cb.getListenersFromIngress(ingress, cbCtx.EnvVariables)

		It("test was setup correctly", func() {
			Expect(ingress.Spec.TLS).ToNot(BeNil())
			Expect(ingress.Annotations[annotations.SslRedirectKey]).To(Equal("true"))
		})

		It("should have created correct ApplicationGatewayRedirectConfiguration struct", func() {
			Expect(len(*actualRedirects)).To(Equal(1))
			Expect(*actualRedirects).To(ContainElement(expectedRedirect))
			Expect(len(actualListeners)).To(Equal(2))
			Expect(actualListeners[listenerID1]).To(Equal(expectedListenerConfigs[listenerID1]))
			Expect(actualListeners[listenerID2]).To(Equal(expectedListenerConfigs[listenerID2]))
			expected := "sslr-fl-bye.com-443"
			Expect(actualListeners[listenerID2].SslRedirectConfigurationName).To(Equal(expected), fmt.Sprintf("Actual: %+v", actualListeners))
		})
	})

	Context("Test RequestRoutingRules without TLS but with SSL Redirect Annotation", func() {
		cb := newConfigBuilderFixture(nil)
		ingress := tests.NewIngressFixture()
		ingress.Spec.TLS = nil
		ingressList := []*v1beta1.Ingress{ingress}
		cbCtx := ConfigBuilderContext{
			IngressList: ingressList,
		}
		actualRedirects := cb.getRedirectConfigurations(&cbCtx)

		// Run this to link the listeners and the redirect config
		actualListeners := cb.getListenersFromIngress(ingress, cbCtx.EnvVariables)

		It("test was setup correctly", func() {
			Expect(ingress.Spec.TLS).To(BeNil())
			Expect(ingress.Annotations[annotations.SslRedirectKey]).To(Equal("true"))
		})

		It("should have created correct ApplicationGatewayRedirectConfiguration struct", func() {
			Expect(len(*actualRedirects)).To(Equal(0))
			Expect(len(actualListeners)).To(Equal(1))
			Expect(actualListeners[listenerID1]).To(Equal(expectedListenerConfigs[listenerID1]), fmt.Sprintf("Actual: %+v", actualListeners))
			Expect(actualListeners[listenerID1].SslRedirectConfigurationName).To(Equal(""), fmt.Sprintf("Actual: %+v", actualListeners))
		})
	})

	Context("Test RequestRoutingRules with TLS but without SSL Redirect Annotation", func() {
		cb := newConfigBuilderFixture(nil)
		ingress := tests.NewIngressFixture()
		delete(ingress.Annotations, annotations.SslRedirectKey)
		ingressList := []*v1beta1.Ingress{ingress}
		cbCtx := ConfigBuilderContext{
			IngressList: ingressList,
		}
		actualRedirects := cb.getRedirectConfigurations(&cbCtx)

		// Run this to link the listeners and the redirect config
		actualListeners := cb.getListenersFromIngress(ingress, cbCtx.EnvVariables)

		It("test was setup correctly", func() {
			Expect(ingress.Spec.TLS).ToNot(BeNil())
			Expect(ingress.Annotations[annotations.SslRedirectKey]).To(Equal(""))
		})

		It("should have created correct ApplicationGatewayRedirectConfiguration struct", func() {
			// Obviously there should be NO redirects since the annotation has been removed
			Expect(len(*actualRedirects)).To(Equal(0))
			Expect(len(actualListeners)).To(Equal(1))
			expectedListenerConfig := listenerAzConfig{
				Protocol: "Https",
				Secret: secretIdentifier{
					Namespace: tests.Namespace,
					Name:      "--the-name-of-the-secret--",
				},
			}
			Expect(actualListeners[listenerID2]).To(Equal(expectedListenerConfig), fmt.Sprintf("Actual: %+v", actualListeners))
			Expect(actualListeners[listenerID2].SslRedirectConfigurationName).To(Equal(""), fmt.Sprintf("Actual: %+v", actualListeners))
		})
	})
})
