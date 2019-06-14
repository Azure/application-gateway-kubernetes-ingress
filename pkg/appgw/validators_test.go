// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/record"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Test ConfigBuilder validator functions", func() {
	Context("test validateURLPathMaps", func() {

		eventRecorder := record.NewFakeRecorder(100)
		ingressList := []*v1beta1.Ingress{}
		serviceList := []*v1.Service{}
		envVariables := environment.GetFakeEnv()

		config := n.ApplicationGatewayPropertiesFormat{
			URLPathMaps: &[]n.ApplicationGatewayURLPathMap{},
		}

		It("", func() {
			err := validateURLPathMaps(eventRecorder, &config, envVariables, ingressList, serviceList)
			Expect(err).To(BeNil())
		})

		It("should error out when no defaults have been set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   nil,
					DefaultBackendAddressPool:    nil,
					DefaultRedirectConfiguration: nil,
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, &config, envVariables, ingressList, serviceList)
			Expect(err).ToNot(BeNil())
			Expect(err).To(Equal(validationErrors[errKeyNoDefaults]))
		})

		It("should error out when all defaults have been set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   &n.SubResource{ID: to.StringPtr("x")},
					DefaultBackendAddressPool:    &n.SubResource{ID: to.StringPtr("x")},
					DefaultRedirectConfiguration: &n.SubResource{ID: to.StringPtr("x")},
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, &config, envVariables, ingressList, serviceList)
			Expect(err).ToNot(BeNil())
			Expect(err).To(Equal(validationErrors[errKeyEitherDefaults]))
		})

		It("should error out when all defaults are partially set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   nil,
					DefaultBackendAddressPool:    &n.SubResource{ID: to.StringPtr("x")},
					DefaultRedirectConfiguration: nil,
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, &config, envVariables, ingressList, serviceList)
			Expect(err).ToNot(BeNil())
			Expect(err).To(Equal(validationErrors[errKeyNoDefaults]))
		})

		It("should NOT error out when all defaults are properly set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   &n.SubResource{ID: to.StringPtr("x")},
					DefaultBackendAddressPool:    &n.SubResource{ID: to.StringPtr("x")},
					DefaultRedirectConfiguration: nil,
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, &config, envVariables, ingressList, serviceList)
			Expect(err).To(BeNil())
		})

		It("should NOT error out when all defaults are properly set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   nil,
					DefaultBackendAddressPool:    nil,
					DefaultRedirectConfiguration: &n.SubResource{ID: to.StringPtr("x")},
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, &config, envVariables, ingressList, serviceList)
			Expect(err).To(BeNil())
		})
	})
})
