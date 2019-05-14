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

	Context("test string key generator too long", func() {
		It("preserves keys of length 80 characters or less", func() {
			actual := governor("this-is-the-key")
			expected := "this-is-the-key"
			Expect(actual).To(Equal(expected))
		})
		It("preserves 80 characters", func() {
			key80Chars := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" +
				"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
			Expect(len(key80Chars)).To(Equal(80))
			actual := governor(key80Chars)
			Expect(actual).To(Equal(key80Chars))
		})
		It("hashes 81 characters", func() {
			key80Chars := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" +
				"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
			Expect(len(key80Chars)).To(Equal(81))
			expected := "xxxxxxxxxxxxxxx-4d1f7bb876c6aae013f06a7430b8c6545ff30d8f9252838eb18b6357c1a2ba13"
			actual := governor(key80Chars)
			Expect(actual).To(Equal(expected))
			Expect(len(actual)).To(Equal(80))
		})
		It("generateProbeName preserves keys in 80 charaters of length or less", func() {
			expected := "k8s-ag-ingress-xxxxxx-yyyyyy-pb-zzzz"
			serviceName := "xxxxxx"
			servicePort := "yyyyyy"
			ingress := "zzzz"
			actual := generateProbeName(serviceName, servicePort, ingress)
			Expect(actual).To(Equal(expected))
		})
		It("generateProbeName relies on governor and hashes long keys", func() {
			expected := "k8s-ag-ingress--9cd4659f054843cb25d7ecb38b0626ce91f0dfbf5099b7f65435b0bd242fd0c0"
			serviceName := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
			servicePort := "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
			ingress := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
			actual := generateProbeName(serviceName, servicePort, ingress)
			Expect(len(actual)).To(Equal(80))
			Expect(actual).To(Equal(expected))
		})
	})
})
