// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
)

func TestHealthProbes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up App Gateway health probes")
}

var _ = Describe("configure App Gateway health probes", func() {
	Context("create probes", func() {
		cb := makeConfigBuilderTestFixture(nil)

		endpoints := makeEndpointsFixture()
		_ = cb.k8sContext.Caches.Endpoints.Add(endpoints)

		service := makeServiceFixture(*makeServicePorts()...)
		_ = cb.k8sContext.Caches.Service.Add(service)

		pod := makePodFixture(testFixturesServiceName, testFixturesNamespace, testFixturesContainerName, testFixturesContainerPort)
		_ = cb.k8sContext.Caches.Pods.Add(pod)

		ingressList := []*v1beta1.Ingress{
			makeIngressFixture(),
		}

		// !! Action !!
		_, _ = cb.HealthProbesCollection(ingressList)
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
			Name: to.StringPtr("k8s-ag-ingress-defaultprobe"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}
		probeForHost := network.ApplicationGatewayProbe{
			ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            network.HTTP,
				Host:                                to.StringPtr(testFixturesHost),
				Path:                                to.StringPtr(testFixturesURLPath),
				Interval:                            to.Int32Ptr(30),
				Timeout:                             to.Int32Ptr(30),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr("k8s-ag-ingress---service-name---443-pb---name--"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}

		probeForOtherHost := network.ApplicationGatewayProbe{
			ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            network.HTTP,
				Host:                                to.StringPtr(testFixturesHost),
				Path:                                to.StringPtr(testFixturesURLPath),
				Interval:                            to.Int32Ptr(20),
				Timeout:                             to.Int32Ptr(5),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr("k8s-ag-ingress---service-name---80-pb---name--"),
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
})
