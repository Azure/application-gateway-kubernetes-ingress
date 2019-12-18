// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Test string key generators", func() {
	veryLongString := "ABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZABCDEFGHIJKLMNOPQRSTUVWXYZAB"
	targetListener := listenerIdentifier{
		FrontendPort: Port(8080),
		HostName:     "foo.baz",
	}
	targetListenerHashCode := utils.GetHashCode(targetListener)

	Context("test each string key generator", func() {
		backendPortNo := Port(8989)
		servicePort := tests.ServicePort
		serviceName := tests.ServiceName
		ingress := tests.NewIngressFixture()
		ingress.Name = "INGR"
		fel := listenerIdentifier{
			FrontendPort: Port(9898),
			HostName:     tests.Host,
		}
		felHashCode := utils.GetHashCode(fel)

		It("getResourceKey returns expected key", func() {
			actual := getResourceKey(tests.Namespace, tests.Name)
			expected := tests.Namespace + "/" + tests.Name
			Expect(actual).To(Equal(expected))
		})

		It("generateHTTPSettingsName returns expected key", func() {
			actual := generateHTTPSettingsName(serviceName, servicePort, backendPortNo, ingress.Name)
			expected := agPrefix + "bp-" + tests.ServiceName + "-" + tests.ServicePort + "-8989-INGR"
			Expect(actual).To(Equal(expected))
		})

		It("generateProbeName returns expected key", func() {
			actual := generateProbeName(serviceName, servicePort, ingress)
			expected := agPrefix + "pb-" + tests.Namespace + "-" + tests.ServiceName + "-" + tests.ServicePort + "-INGR"
			Expect(actual).To(Equal(expected))
		})

		It("generateAddressPoolName returns expected key", func() {
			actual := generateAddressPoolName(serviceName, servicePort, backendPortNo)
			expected := agPrefix + "pool-" + tests.ServiceName + "-" + tests.ServicePort + "-bp-8989"
			Expect(actual).To(Equal(expected))
		})

		It("generateFrontendPortName returns expected key", func() {
			actual := generateFrontendPortName(Port(8989))
			expected := agPrefix + "fp-8989"
			Expect(actual).To(Equal(expected))
		})

		It("generateListenerName returns expected key", func() {
			actual := generateListenerName(fel)
			expected := agPrefix + "fl-" + felHashCode
			Expect(actual).To(Equal(expected))
		})

		It("generateURLPathMapName returns expected key", func() {
			actual := generateURLPathMapName(fel)
			expected := agPrefix + "url-" + felHashCode
			Expect(actual).To(Equal(expected))
		})

		It("generateRequestRoutingRuleName returns expected key", func() {
			actual := generateRequestRoutingRuleName(fel)
			expected := agPrefix + "rr-" + felHashCode
			Expect(actual).To(Equal(expected))
		})

		It("generateSSLRedirectConfigurationName returns expected key", func() {
			actual := generateSSLRedirectConfigurationName(targetListener)
			expected := "sslr-fl-" + targetListenerHashCode
			Expect(actual).To(Equal(expected))
		})
	})

	Context("test string key generator with long strings", func() {
		It("should create correct keys when these are over 80 characters long", func() {
			actual := formatPropName("this-is-the-key")
			expected := "this-is-the-key"
			Expect(actual).To(Equal(expected), fmt.Sprintf("Expected name: %s", expected))
		})
		It("preserves 80 characters", func() {
			key80Chars := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" +
				"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
			Expect(len(key80Chars)).To(Equal(80))
			actual := formatPropName(key80Chars)
			Expect(actual).To(Equal(key80Chars))
		})
		It("hashes 81 characters", func() {
			key80Chars := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" +
				"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
			Expect(len(key80Chars)).To(Equal(81))
			expected := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-21360fb332fac3e20710495d135a23d4"
			actual := formatPropName(key80Chars)
			Expect(actual).To(Equal(expected))
			Expect(len(actual)).To(Equal(80))
		})
		It("generateProbeName preserves keys in 80 charaters of length or less", func() {
			expected := agPrefix + "pb-" + tests.Namespace + "-xxxxxx-yyyyyy-zzzz"
			serviceName := "xxxxxx"
			servicePort := "yyyyyy"
			ingress := tests.NewIngressFixture()
			ingress.Name = "zzzz"
			actual := generateProbeName(serviceName, servicePort, ingress)
			Expect(actual).To(Equal(expected))
		})
		It("generateProbeName relies on formatPropName and hashes long keys", func() {
			expected := "pb-" + tests.Namespace + "-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-a95496d0f08ce51d96b204851450bf64"
			serviceName := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
			servicePort := "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
			ingress := tests.NewIngressFixture()
			ingress.Name = "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
			actual := generateProbeName(serviceName, servicePort, ingress)
			Expect(len(actual)).To(Equal(80))
			Expect(actual).To(Equal(expected), fmt.Sprintf("Expected %s; Got %s", expected, actual))
		})
	})

	Context("test property names that require a host name, but host name is blank", func() {
		// Listener without a hostname
		listener := listenerIdentifier{
			FrontendPort: Port(9898),
		}

		listenerName := generateListenerName(listener)
		It("generateListenerName should have generated correct name without host name", func() {
			Expect(listenerName).To(Equal("fl-" + utils.GetHashCode(listener)))
		})

		pathMapName := generateURLPathMapName(listener)
		It("generateURLPathMapName should have generated correct name without host name", func() {
			Expect(pathMapName).To(Equal("url-" + utils.GetHashCode(listener)))
		})

		ruleName := generateRequestRoutingRuleName(listener)
		It("generateRequestRoutingRuleName should have generated correct name without host name", func() {
			Expect(ruleName).To(Equal("rr-" + utils.GetHashCode(listener)))
		})
	})

	Context("test property name generator with very long strings", func() {
		namespace := veryLongString
		name := veryLongString
		serviceName := veryLongString
		servicePort := veryLongString
		backendPortNo := Port(8888)
		ingress := tests.NewIngressFixture()
		ingress.Name = veryLongString
		port := Port(88)
		felID := listenerIdentifier{
			FrontendPort: port,
			HostName:     namespace,
		}
		names := []string{
			getResourceKey(namespace, name),
			generateHTTPSettingsName(serviceName, servicePort, backendPortNo, ingress.Name),
			generateProbeName(serviceName, servicePort, ingress),
			generateAddressPoolName(serviceName, servicePort, backendPortNo),
			generateFrontendPortName(port),
			generateListenerName(felID),
			generateURLPathMapName(felID),
			generateRequestRoutingRuleName(felID),
			generateSSLRedirectConfigurationName(targetListener),
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

	Context("test whether getResourceKey works correctly", func() {
		It("should construct correct key", func() {
			actual := getResourceKey(tests.Namespace, tests.Name)
			expected := tests.Namespace + "/" + tests.Name
			Expect(actual).To(Equal(expected))
		})
	})

	Context("test GetHostNames works correctly", func() {
		It("should correctly return the hostnames", func() {
			var hostnameValues = [5]string{"www.test1.com", "www.test2.com", "www.test3.com","www.test4.com","www.test5.com"}
			listenerID := listenerIdentifier{
				FrontendPort: Port(80),
				UsePrivateIP: false,
				HostName: "www.test.com",
				HostNames: hostnameValues,
			}
			actualHostName := listenerID.getHostNames()
			Expect(actualHostName).To(Equal(hostnameValues[0:]))
		})

		It("should return nil if the 'hostnames' field is not set", func() {
			listenerID := listenerIdentifier{
				FrontendPort: Port(80),
				UsePrivateIP: false,
			}
			actualHostName := listenerID.getHostNames()
			Expect(actualHostName).To(BeNil())
		})
	})

	Context("test SetHostNames works correctly", func() {
		It("should correctly update the listenerIdentifier", func() {
			listenerID := listenerIdentifier{
				FrontendPort: Port(80),
				UsePrivateIP: false,
			}
			hostnames := []string{"www.test.com", "www.t*.com"}
			listenerID.setHostNames(hostnames)
			Expect(listenerID.HostName).To(Equal("www.test.com"))
			Expect(listenerID.HostNames[0]).To(Equal("www.test.com"))
			Expect(listenerID.HostNames[1]).To(Equal("www.t*.com"))
			Expect(listenerID.HostNames[2]).To(Equal(""))
		})
	})
})
