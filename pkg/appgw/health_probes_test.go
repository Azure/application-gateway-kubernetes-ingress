// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("configure App Gateway health probes", func() {
	Context("create probes", func() {
		cb := newConfigBuilderFixture(nil)

		endpoints := tests.NewEndpointsFixture()
		_ = cb.k8sContext.Caches.Endpoints.Add(endpoints)

		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		_ = cb.k8sContext.Caches.Service.Add(service)

		pod := tests.NewPodFixture(tests.ServiceName, tests.Namespace, tests.ContainerName, tests.ContainerPort)
		_ = cb.k8sContext.Caches.Pods.Add(pod)

		ingressList := []*v1beta1.Ingress{
			tests.NewIngressFixture(),
		}

		// !! Action !!
		_ = cb.HealthProbesCollection(ingressList)
		actual := cb.appGwConfig.Probes

		// We expect our health probe configurator to have arrived at this final setup
		defaultProbe := network.ApplicationGatewayProbe{

			ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            network.HTTP,
				Host:                                to.StringPtr("localhost"),
				Path:                                to.StringPtr("/"),
				Interval:                            to.Int32Ptr(30),
				Timeout:                             to.Int32Ptr(30),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr(agPrefix + "defaultprobe"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}
		probeForHost := network.ApplicationGatewayProbe{
			ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            network.HTTP,
				Host:                                to.StringPtr(tests.Host),
				Path:                                to.StringPtr(tests.URLPath),
				Interval:                            to.Int32Ptr(30),
				Timeout:                             to.Int32Ptr(30),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr(agPrefix + "pb---service-name---443---name--"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}

		probeForOtherHost := network.ApplicationGatewayProbe{
			ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            network.HTTP,
				Host:                                to.StringPtr(tests.Host),
				Path:                                to.StringPtr(tests.URLPath),
				Interval:                            to.Int32Ptr(20),
				Timeout:                             to.Int32Ptr(5),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr(agPrefix + "pb---service-name---80---name--"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}

		It("should have exactly 3 records", func() {
			Expect(len(*actual)).To(Equal(3))
		})

		It("should have created 1 default probe", func() {
			Expect(*actual).To(ContainElement(defaultProbe))
		})

		It("should have created 1 probe for Host", func() {
			Expect(*actual).To(ContainElement(probeForHost))
		})

		It("should have created 1 probe for OtherHost", func() {
			Expect(*actual).To(ContainElement(probeForOtherHost))
		})
	})

	Context("use default probe when service doesn't exists", func() {
		cb := newConfigBuilderFixture(nil)

		pod := tests.NewPodFixture(tests.ServiceName, tests.Namespace, tests.ContainerName, tests.ContainerPort)
		_ = cb.k8sContext.Caches.Pods.Add(pod)

		ingressList := []*v1beta1.Ingress{
			tests.NewIngressFixture(),
		}

		// !! Action !!
		_ = cb.HealthProbesCollection(ingressList)
		actual := cb.appGwConfig.Probes

		// We expect our health probe configurator to have arrived at this final setup
		defaultProbe := network.ApplicationGatewayProbe{

			ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            network.HTTP,
				Host:                                to.StringPtr("localhost"),
				Path:                                to.StringPtr("/"),
				Interval:                            to.Int32Ptr(30),
				Timeout:                             to.Int32Ptr(30),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr(agPrefix + "defaultprobe"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}

		It("should have exactly 1 record", func() {
			Expect(len(*actual)).To(Equal(1))
		})

		It("should have created 1 default probe", func() {
			Expect(*actual).To(ContainElement(defaultProbe))
		})
	})
})
