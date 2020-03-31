// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Test ConfigBuilder validator functions", func() {
	Context("test validateURLPathMaps", func() {

		eventRecorder := record.NewFakeRecorder(100)
		ingressList := []*v1beta1.Ingress{}
		serviceList := []*v1.Service{}
		envVariables := environment.GetFakeEnv()

		config := &n.ApplicationGatewayPropertiesFormat{
			URLPathMaps: &[]n.ApplicationGatewayURLPathMap{},
		}

		It("", func() {
			err := validateURLPathMaps(eventRecorder, config, envVariables, ingressList, serviceList)
			Expect(err).To(BeNil())
		})

		It("should error out when no defaults have been set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				Name: to.StringPtr("pathMap"),
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   nil,
					DefaultBackendAddressPool:    nil,
					DefaultRedirectConfiguration: nil,
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, config, envVariables, ingressList, serviceList)
			Expect(err).ToNot(BeNil())
			Expect(err.(*controllererrors.Error).Code).To(Equal(controllererrors.ErrorNoDefaults))
		})

		It("should error out when all defaults have been set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				Name: to.StringPtr("pathMap"),
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   &n.SubResource{ID: to.StringPtr("x")},
					DefaultBackendAddressPool:    &n.SubResource{ID: to.StringPtr("x")},
					DefaultRedirectConfiguration: &n.SubResource{ID: to.StringPtr("x")},
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, config, envVariables, ingressList, serviceList)
			Expect(err).ToNot(BeNil())
			Expect(err.(*controllererrors.Error).Code).To(Equal(controllererrors.ErrorEitherDefaults))
		})

		It("should error out when all defaults are partially set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				Name: to.StringPtr("pathMap"),
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   nil,
					DefaultBackendAddressPool:    &n.SubResource{ID: to.StringPtr("x")},
					DefaultRedirectConfiguration: nil,
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, config, envVariables, ingressList, serviceList)
			Expect(err).ToNot(BeNil())
			Expect(err.(*controllererrors.Error).Code).To(Equal(controllererrors.ErrorNoDefaults))
		})

		It("should NOT error out when all defaults are properly set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				Name: to.StringPtr("pathMap"),
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   &n.SubResource{ID: to.StringPtr("x")},
					DefaultBackendAddressPool:    &n.SubResource{ID: to.StringPtr("x")},
					DefaultRedirectConfiguration: nil,
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, config, envVariables, ingressList, serviceList)
			Expect(err).To(BeNil())
		})

		It("should NOT error out when all defaults are properly set", func() {
			pathMap := n.ApplicationGatewayURLPathMap{
				Name: to.StringPtr("pathMap"),
				ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules:                    &[]n.ApplicationGatewayPathRule{},
					DefaultBackendHTTPSettings:   nil,
					DefaultBackendAddressPool:    nil,
					DefaultRedirectConfiguration: &n.SubResource{ID: to.StringPtr("x")},
				},
			}
			config.URLPathMaps = &[]n.ApplicationGatewayURLPathMap{pathMap}
			err := validateURLPathMaps(eventRecorder, config, envVariables, ingressList, serviceList)
			Expect(err).To(BeNil())
		})
	})

	Context("test validateFrontendIPConfiguration", func() {
		eventRecorder := record.NewFakeRecorder(100)
		envVariables := environment.GetFakeEnv()

		publicIPConf := n.ApplicationGatewayFrontendIPConfiguration{
			// Public IP
			Name: to.StringPtr("xx3"),
			Etag: to.StringPtr("xx2"),
			Type: to.StringPtr("xx1"),
			ID:   to.StringPtr(tests.PublicIPID),
			ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &n.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
				PrivateIPAddress: nil,
				PublicIPAddress: &n.SubResource{
					ID: to.StringPtr("xyz"),
				},
			},
		}

		privateIPConf := n.ApplicationGatewayFrontendIPConfiguration{
			// Private IP
			Name: to.StringPtr("yy3"),
			Etag: to.StringPtr("yy2"),
			Type: to.StringPtr("yy1"),
			ID:   to.StringPtr(tests.PrivateIPID),
			ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &n.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
				PrivateIPAddress: to.StringPtr("abc"),
				PublicIPAddress:  nil,
			},
		}

		config := n.ApplicationGatewayPropertiesFormat{
			FrontendIPConfigurations: &[]n.ApplicationGatewayFrontendIPConfiguration{},
		}

		It("should error out when Ip Configuration is empty.", func() {
			config.FrontendIPConfigurations = &[]n.ApplicationGatewayFrontendIPConfiguration{}
			err := validateFrontendIPConfiguration(eventRecorder, config, envVariables)
			Expect(err.(*controllererrors.Error).Code).To(Equal(controllererrors.ErrorNoPublicIP))
		})

		It("should not error out when Ip Configuration is contains 1 PublicIP and UsePrivateIP is false.", func() {
			config.FrontendIPConfigurations = &[]n.ApplicationGatewayFrontendIPConfiguration{publicIPConf}
			err := validateFrontendIPConfiguration(eventRecorder, config, envVariables)
			Expect(err).To(BeNil())
		})

		It("should not error out when Ip Configuration is contains both PublicIP & PrivateIP and UsePrivateIP is false.", func() {
			config.FrontendIPConfigurations = &[]n.ApplicationGatewayFrontendIPConfiguration{publicIPConf, privateIPConf}
			err := validateFrontendIPConfiguration(eventRecorder, config, envVariables)
			Expect(err).To(BeNil())
		})

		It("should not error out when Ip Configuration is contains both PublicIP & PrivateIP and UsePrivateIP is true.", func() {
			envVariablesNew := environment.GetFakeEnv()
			envVariablesNew.UsePrivateIP = "true"
			Expect(envVariablesNew.UsePrivateIP).To(Equal("true"))
			config.FrontendIPConfigurations = &[]n.ApplicationGatewayFrontendIPConfiguration{publicIPConf, privateIPConf}
			err := validateFrontendIPConfiguration(eventRecorder, config, envVariablesNew)
			Expect(err).To(BeNil())
		})

		It("should error out when Ip Configuration is contains 1 PublicIP and UsePrivateIP is true.", func() {
			envVariablesNew := environment.GetFakeEnv()
			envVariablesNew.UsePrivateIP = "true"
			Expect(envVariablesNew.UsePrivateIP).To(Equal("true"))
			config.FrontendIPConfigurations = &[]n.ApplicationGatewayFrontendIPConfiguration{publicIPConf}
			err := validateFrontendIPConfiguration(eventRecorder, config, envVariablesNew)
			Expect(err.(*controllererrors.Error).Code).To(Equal(controllererrors.ErrorNoPrivateIP))

		})

		It("should error out when Ip Configuration is doesn't contain public IP.", func() {
			config.FrontendIPConfigurations = &[]n.ApplicationGatewayFrontendIPConfiguration{privateIPConf}
			err := validateFrontendIPConfiguration(eventRecorder, config, envVariables)
			Expect(err.(*controllererrors.Error).Code).To(Equal(controllererrors.ErrorNoPublicIP))
		})
	})
})
