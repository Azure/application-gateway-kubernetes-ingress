// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStringKeyGenerators(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up App Gateway health probes")
}

var _ = Describe("Test string key generators", func() {
	Context("test each string key generator", func() {
		backendPortNo := int32(8989)
		servicePort := "-service-port-"
		serviceName := "--service--name--"
		ingress := "---ingress---"
		fel := frontendListenerIdentifier{
			FrontendPort: int32(9898),
			HostName:     "--host--name--",
		}

		It("getResourceKey returns expected key", func() {
			actual := getResourceKey(testFixturesNamespace, testFixturesName)
			expected := "--namespace--/--name--"
			Expect(actual).To(Equal(expected))
		})

		It("generateHTTPSettingsName returns expected key", func() {
			actual := generateHTTPSettingsName(serviceName, servicePort, backendPortNo, ingress)
			expected := "k8s-ag-ingress---service--name----service-port--bp-8989----ingress---"
			Expect(actual).To(Equal(expected))
		})

		It("generateProbeName returns expected key", func() {
			actual := generateProbeName(serviceName, servicePort, ingress)
			expected := "k8s-ag-ingress---service--name----service-port--pb----ingress---"
			Expect(actual).To(Equal(expected))
		})

		It("generateAddressPoolName returns expected key", func() {
			actual := generateAddressPoolName(serviceName, servicePort, backendPortNo)
			expected := "k8s-ag-ingress---service--name----service-port--bp-8989-pool"
			Expect(actual).To(Equal(expected))
		})

		It("generateFrontendPortName returns expected key", func() {
			actual := generateFrontendPortName(int32(8989))
			expected := "k8s-ag-ingress-fp-8989"
			Expect(actual).To(Equal(expected))
		})

		It("generateHTTPListenerName returns expected key", func() {
			actual := generateHTTPListenerName(fel)
			expected := "k8s-ag-ingress---host--name---9898-fl"
			Expect(actual).To(Equal(expected))
		})

		It("generateURLPathMapName returns expected key", func() {
			actual := generateURLPathMapName(fel)
			expected := "k8s-ag-ingress---host--name---9898-url"
			Expect(actual).To(Equal(expected))
		})

		It("generateRequestRoutingRuleName returns expected key", func() {
			actual := generateRequestRoutingRuleName(fel)
			expected := "k8s-ag-ingress---host--name---9898-rr"
			Expect(actual).To(Equal(expected))
		})

		It("generateSSLRedirectConfigurationName returns expected key", func() {
			actual := generateSSLRedirectConfigurationName(testFixturesNamespace, ingress)
			expected := "k8s-ag-ingress---namespace------ingress----sslr"
			Expect(actual).To(Equal(expected))
		})
	})
})
