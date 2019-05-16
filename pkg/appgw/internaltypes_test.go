// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStringKeyGenerators(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up App Gateway health probes")
}

var _ = Describe("Test string key generators", func() {
	veryLongString := "ABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZAB"

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
			expected := agPrefix + "--service--name----service-port--bp-8989----ingress---"
			Expect(actual).To(Equal(expected))
		})

		It("generateProbeName returns expected key", func() {
			actual := generateProbeName(serviceName, servicePort, ingress)
			expected := agPrefix + "--service--name----service-port--pb----ingress---"
			Expect(actual).To(Equal(expected))
		})

		It("generateAddressPoolName returns expected key", func() {
			actual := generateAddressPoolName(serviceName, servicePort, backendPortNo)
			expected := agPrefix + "--service--name----service-port--bp-8989-pool"
			Expect(actual).To(Equal(expected))
		})

		It("generateFrontendPortName returns expected key", func() {
			actual := generateFrontendPortName(int32(8989))
			expected := agPrefix + "fp-8989"
			Expect(actual).To(Equal(expected))
		})

		It("generateHTTPListenerName returns expected key", func() {
			actual := generateHTTPListenerName(fel)
			expected := agPrefix + "--host--name---9898-fl"
			Expect(actual).To(Equal(expected))
		})

		It("generateURLPathMapName returns expected key", func() {
			actual := generateURLPathMapName(fel)
			expected := agPrefix + "--host--name---9898-url"
			Expect(actual).To(Equal(expected))
		})

		It("generateRequestRoutingRuleName returns expected key", func() {
			actual := generateRequestRoutingRuleName(fel)
			expected := agPrefix + "--host--name---9898-rr"
			Expect(actual).To(Equal(expected))
		})

		It("generateSSLRedirectConfigurationName returns expected key", func() {
			actual := generateSSLRedirectConfigurationName(testFixturesNamespace, ingress)
			expected := agPrefix + "--namespace------ingress----sslr"
			Expect(actual).To(Equal(expected))
		})
	})

	Context("test string key generator with long strings", func() {
		It("should create correct keys when these are over 80 characters long", func() {
			actual := governor("this-is-the-key")
			expected := "this-is-the-key"
			Expect(actual).To(Equal(expected), fmt.Sprintf("Expected name: %s", expected))
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
			expected := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-21360fb332fac3e20710495d135a23d4"
			actual := governor(key80Chars)
			Expect(actual).To(Equal(expected))
			Expect(len(actual)).To(Equal(80))
		})
		It("generateProbeName preserves keys in 80 charaters of length or less", func() {
			expected := "xxxxxx-yyyyyy-pb-zzzz"
			serviceName := "xxxxxx"
			servicePort := "yyyyyy"
			ingress := "zzzz"
			actual := generateProbeName(serviceName, servicePort, ingress)
			Expect(actual).To(Equal(expected))
		})
		It("generateProbeName relies on governor and hashes long keys", func() {
			expected := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-yyyy-f6ec0d78e6dd6290971a7de1ed28e6a1"
			serviceName := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
			servicePort := "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
			ingress := "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
			actual := generateProbeName(serviceName, servicePort, ingress)
			Expect(len(actual)).To(Equal(80))
			Expect(actual).To(Equal(expected), fmt.Sprintf("Expected %s; Got %s", expected, actual))
		})
	})

	Context("test property name generator with very long strings", func() {
		namespace := veryLongString
		name := veryLongString
		serviceName := veryLongString
		servicePort := veryLongString
		backendPortNo := int32(8888)
		ingress := veryLongString
		port := int32(88)
		felID := frontendListenerIdentifier{
			FrontendPort: port,
			HostName:     namespace,
		}
		names := []string{
			getResourceKey(namespace, name),
			generateHTTPSettingsName(serviceName, servicePort, backendPortNo, ingress),
			generateProbeName(serviceName, servicePort, ingress),
			generateAddressPoolName(serviceName, servicePort, backendPortNo),
			generateFrontendPortName(port),
			generateHTTPListenerName(felID),
			generateURLPathMapName(felID),
			generateRequestRoutingRuleName(felID),
			generateSSLRedirectConfigurationName(namespace, ingress),
		}
		It("ensure test is setup correctly", func() {
			// ensure this is setup correctly
			Ω(len(veryLongString)).Should(BeNumerically(">=", 80))
			Expect(len(names)).To(Equal(9))
		})
		It("should ensure that all names generated are no longer than 80 characters and are unique", func() {
			// Ensure the strings are unique
			nameSet := make(map[string]interface{})
			for _, name := range names {
				Ω(len(name)).Should(BeNumerically("<=", 80))
				nameSet[name] = nil
			}
			// Uniqueness test
			Expect(len(nameSet)).To(Equal(len(names)))
		})
	})

	Context("test agPrefix sanitizer", func() {
		It("should fail for long strings", func() {
			// ensure this is setup correctly
			Expect(agPrefixValidator.MatchString(veryLongString)).To(BeFalse())
		})
		It("should pass for short alphanumeric strings", func() {
			// ensure this is setup correctly
			Expect(agPrefixValidator.MatchString("abc-xyz")).To(BeTrue())
		})
		It("should pass for empty strings", func() {
			// ensure this is setup correctly
			Expect(agPrefixValidator.MatchString("")).To(BeTrue())
		})
		It("should fail for non alphanumeric strings", func() {
			// ensure this is setup correctly
			Expect(agPrefixValidator.MatchString("omega----Ω")).To(BeFalse())
		})
	})
})
