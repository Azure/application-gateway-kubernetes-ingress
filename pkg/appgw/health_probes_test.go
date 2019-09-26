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
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("configure App Gateway health probes", func() {
	ingressList := []*v1beta1.Ingress{tests.NewIngressFixture()}
	serviceList := []*v1.Service{tests.NewServiceFixture()}

	Context("create probes", func() {
		cb := newConfigBuilderFixture(nil)

		endpoints := tests.NewEndpointsFixture()
		_ = cb.k8sContext.Caches.Endpoints.Add(endpoints)

		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		_ = cb.k8sContext.Caches.Service.Add(service)

		pod := tests.NewPodFixture(tests.ServiceName, tests.Namespace, tests.ContainerName, tests.ContainerPort)
		_ = cb.k8sContext.Caches.Pods.Add(pod)

		cbCtx := &ConfigBuilderContext{
			IngressList: ingressList,
			ServiceList: serviceList,
		}

		// !! Action !!
		_ = cb.HealthProbesCollection(cbCtx)
		actual := cb.appGw.Probes

		// We expect our health probe configurator to have arrived at this final setup
		probeName := agPrefix + "pb-" + tests.Namespace + "-" + tests.ServiceName + "-443---name--"
		probeForHost := n.ApplicationGatewayProbe{
			ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            n.HTTP,
				Host:                                to.StringPtr(tests.Host),
				Path:                                to.StringPtr(tests.HealthPath),
				Interval:                            to.Int32Ptr(20),
				Timeout:                             to.Int32Ptr(5),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
				Port:                                to.Int32Ptr(9090),
			},
			Name: to.StringPtr(probeName),
			Etag: nil,
			Type: nil,
			ID:   to.StringPtr(cb.appGwIdentifier.probeID(probeName)),
		}

		probeName = agPrefix + "pb-" + tests.Namespace + "-" + tests.ServiceName + "-80---name--"
		probeForOtherHost := n.ApplicationGatewayProbe{
			ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            n.HTTP,
				Host:                                to.StringPtr(tests.Host),
				Path:                                to.StringPtr(tests.HealthPath),
				Interval:                            to.Int32Ptr(20),
				Timeout:                             to.Int32Ptr(5),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
				Port:                                to.Int32Ptr(9090),
			},
			Name: to.StringPtr(probeName),
			Etag: nil,
			Type: nil,
			ID:   to.StringPtr(cb.appGwIdentifier.probeID(probeName)),
		}

		It("should have exactly 4 records", func() {
			Expect(len(*actual)).To(Equal(4))
		})

		It("should have created 1 default probe", func() {
			Expect(*actual).To(ContainElement(defaultProbe(cb.appGwIdentifier, n.HTTP)))
		})

		It("should have created 1 probe for Host", func() {
			Expect(*actual).To(ContainElement(probeForHost))
		})

		It("should have created 1 probe for OtherHost", func() {
			Expect(*actual).To(ContainElement(probeForOtherHost))
		})
	})

	Context("respect liveness/readiness probe even when backend protocol is http on ingress", func() {
		cb := newConfigBuilderFixture(nil)

		endpoints := tests.NewEndpointsFixture()
		_ = cb.k8sContext.Caches.Endpoints.Add(endpoints)

		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		_ = cb.k8sContext.Caches.Service.Add(service)

		pod := tests.NewPodFixture(tests.ServiceName, tests.Namespace, tests.ContainerName, tests.ContainerPort)
		pod.Spec.Containers[0].ReadinessProbe.HTTPGet.Scheme = v1.URISchemeHTTPS
		_ = cb.k8sContext.Caches.Pods.Add(pod)

		cbCtx := &ConfigBuilderContext{
			IngressList: ingressList,
			ServiceList: serviceList,
		}

		// !! Action !!
		probeMap, _ := cb.newProbesMap(cbCtx)

		backend := ingressList[0].Spec.Rules[0].HTTP.Paths[0].Backend
		probeName := generateProbeName(backend.ServiceName, backend.ServicePort.String(), ingressList[0])
		It("uses the readiness probe to set the protocol on the probe", func() {
			Expect(probeMap[probeName].Protocol).To(Equal(n.HTTPS))
		})
	})

	Context("use default probe when service doesn't exists", func() {
		cb := newConfigBuilderFixture(nil)

		pod := tests.NewPodFixture(tests.ServiceName, tests.Namespace, tests.ContainerName, tests.ContainerPort)
		_ = cb.k8sContext.Caches.Pods.Add(pod)

		cbCtx := &ConfigBuilderContext{
			IngressList: ingressList,
			ServiceList: serviceList,
		}

		// !! Action !!
		_ = cb.HealthProbesCollection(cbCtx)
		actual := cb.appGw.Probes

		It("should have exactly 2 record", func() {
			Expect(len(*actual)).To(Equal(2), fmt.Sprintf("Actual probes: %+v", *actual))
		})

		It("should have created 2 default probes", func() {
			Expect(*actual).To(ContainElement(defaultProbe(cb.appGwIdentifier, n.HTTP)))
			Expect(*actual).To(ContainElement(defaultProbe(cb.appGwIdentifier, n.HTTPS)))
		})
	})

	Context("test generateHealthProbe()", func() {
		cb := newConfigBuilderFixture(nil)
		be := backendIdentifier{
			serviceIdentifier: serviceIdentifier{
				Namespace: "default",
				Name:      "blah",
			},
			Ingress: fixtures.GetIngress(),
			Rule:    nil,
			Path:    nil,
			Backend: &v1beta1.IngressBackend{},
		}
		pb := cb.generateHealthProbe(be)
		It("should return nil and not crash", func() {
			Expect(pb).To(BeNil())
		})
	})

})
