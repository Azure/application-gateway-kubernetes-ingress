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
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHealthProbes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test setting up App Gateway health probes")
}

var _ = Describe("configure App Gateway health probes", func() {
	port1, port2, port3, port4 := makeServicePorts()

	Context("looking at TLS specs", func() {
		cb := makeConfigBuilderTestFixture(nil)
		endpoints := v1.Endpoints{
			Subsets: []v1.EndpointSubset{
				{
					// IP addresses which offer the related ports that are marked as ready. These endpoints
					// should be considered safe for load balancers and clients to utilize.
					// +optional
					Addresses: []v1.EndpointAddress{
						{
							IP: "10.9.8.7",
							// The Hostname of this endpoint
							// +optional
							Hostname: "www.contoso.com",
							// Optional: Node hosting this endpoint. This can be used to determine endpoints local to a node.
							// +optional
							NodeName: to.StringPtr(testFixturesNodeName),
						},
					},
					// IP addresses which offer the related ports but are not currently marked as ready
					// because they have not yet finished starting, have recently failed a readiness check,
					// or have recently failed a liveness check.
					// +optional
					NotReadyAddresses: []v1.EndpointAddress{},
					// Port numbers available on the related IP addresses.
					// +optional
					Ports: []v1.EndpointPort{},
				},
			},
		}
		err := cb.k8sContext.Caches.Endpoints.Add(&endpoints)
		It("added endpoints to cache without an error", func() {
			Expect(err).To(BeNil())
		})

		service := v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testFixturesServiceName,
				Namespace: testFixturesNamespace,
			},
			Spec: v1.ServiceSpec{
				// List of ports exposed by this service
				Ports: []v1.ServicePort{
					port1,
					port2,
					port3,
					port4,
				},
			},
		}
		err = cb.k8sContext.Caches.Service.Add(&service)

		backendName := "--backend-name--"
		backendPort := int32(9876)
		pod := makePod(service.Name, testFixturesNamespace, backendName, backendPort)
		err = cb.k8sContext.Caches.Pods.Add(pod)

		It("added service to cache without an error", func() {
			Expect(err).To(BeNil())
		})

		ingress := makeIngressTestFixture()
		ingressList := []*v1beta1.Ingress{
			&ingress,
		}

		// !! Action !!
		_, _ = cb.HealthProbesCollection(ingressList)
		actual := cb.appGwConfig.Probes

		// We expect our health probe configurator to have arrived at this final setup
		expected := []network.ApplicationGatewayProbe{
			{
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
			},
			{
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
				Name: to.StringPtr("k8s-ag-ingress--8080-pb---name--"),
				Etag: nil,
				Type: nil,
				ID:   nil,
			},
			{
				ApplicationGatewayProbePropertiesFormat: &network.ApplicationGatewayProbePropertiesFormat{
					Protocol:                            network.HTTP,
					Host:                                to.StringPtr(testFixturesOtherHost),
					Path:                                to.StringPtr(testFixturesURLPath),
					Interval:                            to.Int32Ptr(30),
					Timeout:                             to.Int32Ptr(30),
					UnhealthyThreshold:                  to.Int32Ptr(3),
					PickHostNameFromBackendHTTPSettings: nil,
					MinServers:                          nil,
					Match:                               nil,
					ProvisioningState:                   nil,
				},
				Name: to.StringPtr("k8s-ag-ingress--8989-pb---name--"),
				Etag: nil,
				Type: nil,
				ID:   nil,
			},
		}

		It("should have exactly 3 records", func() {
			Expect(len(*actual)).To(Equal(3))
		})

		It("should succeed", func() {
			// Ensure capacities of the slices match
			Expect(*actual).To(ContainElement(expected[0]))
			Expect(*actual).To(ContainElement(expected[1]))
			Expect(*actual).To(ContainElement(expected[2]))
		})
	})
})
