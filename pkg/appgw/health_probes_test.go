// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("configure App Gateway health probes", func() {
	ingressList := []*networking.Ingress{tests.NewIngressFixture()}
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
			IngressList:           ingressList,
			ServiceList:           serviceList,
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
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
				Match:                               &n.ApplicationGatewayProbeHealthResponseMatch{},
				PickHostNameFromBackendHTTPSettings: to.BoolPtr(false),
				MinServers:                          to.Int32Ptr(0),
				ProvisioningState:                   "",
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
				Match:                               &n.ApplicationGatewayProbeHealthResponseMatch{},
				PickHostNameFromBackendHTTPSettings: to.BoolPtr(false),
				MinServers:                          to.Int32Ptr(0),
				ProvisioningState:                   "",
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
			IngressList:           ingressList,
			ServiceList:           serviceList,
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		// !! Action !!
		probeMap, _ := cb.newProbesMap(cbCtx)

		backend := ingressList[0].Spec.Rules[0].HTTP.Paths[0].Backend
		probeName := generateProbeName(backend.Service.Name, serviceBackendPortToStr(backend.Service.Port), ingressList[0])
		It("uses the readiness probe to set the protocol on the probe", func() {
			Expect(probeMap[probeName].Protocol).To(Equal(n.HTTPS))
		})
	})

	Context("use default probe when service doesn't exists", func() {
		cb := newConfigBuilderFixture(nil)

		pod := tests.NewPodFixture(tests.ServiceName, tests.Namespace, tests.ContainerName, tests.ContainerPort)
		_ = cb.k8sContext.Caches.Pods.Add(pod)

		cbCtx := &ConfigBuilderContext{
			IngressList:           ingressList,
			ServiceList:           serviceList,
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
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
			Backend: &networking.IngressBackend{},
		}
		pb := cb.generateHealthProbe(be)
		It("should return nil and not crash", func() {
			Expect(pb).To(BeNil())
		})
	})

	Context("ensure that annotation overrides defaults for health probe", func() {

		annotationHpHostname := "myhost.mydomain.com"
		annotationHpPort := int32(8080)
		annotationHpPath := "/healthz"
		annotationHpCodes := "200-399,401"
		annotationHpInterval := int32(15)
		annotationHpTimeout := int32(10)
		annotationHpThreshold := int32(3)
		statusCodes := strings.Split(annotationHpCodes, ",")

		annotations := map[string]string{
			"kubernetes.io/ingress.class":                                  "azure/application-gateway",
			"appgw.ingress.kubernetes.io/health-probe-hostname":            annotationHpHostname,
			"appgw.ingress.kubernetes.io/health-probe-port":                strconv.Itoa(int(annotationHpPort)),
			"appgw.ingress.kubernetes.io/health-probe-path":                annotationHpPath,
			"appgw.ingress.kubernetes.io/health-probe-status-codes":        annotationHpCodes,
			"appgw.ingress.kubernetes.io/health-probe-interval":            strconv.Itoa(int(annotationHpInterval)),
			"appgw.ingress.kubernetes.io/health-probe-timeout":             strconv.Itoa(int(annotationHpTimeout)),
			"appgw.ingress.kubernetes.io/health-probe-unhealthy-threshold": strconv.Itoa(int(annotationHpThreshold)),
		}

		ingress := fixtures.GetIngress()
		ingress.ObjectMeta.Annotations = annotations

		cb := newConfigBuilderFixture(nil)
		be := backendIdentifier{
			serviceIdentifier: serviceIdentifier{
				Namespace: "--namespace--",
				Name:      "--service-name--",
			},
			Ingress: ingress,
			Rule:    nil,
			Path: &networking.HTTPIngressPath{
				Path: "/test",
				Backend: networking.IngressBackend{
					Service: &networking.IngressServiceBackend{
						Name: "--service-name--",
					},
				},
			},
			Backend: &networking.IngressBackend{},
		}
		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		_ = cb.k8sContext.Caches.Service.Add(service)

		pb := cb.generateHealthProbe(be)

		It("probe hostname must match annotation", func() {
			Expect(pb.ApplicationGatewayProbePropertiesFormat.Host).Should(Equal(&annotationHpHostname))
		})
		It("probe port must match annotation", func() {
			Expect(pb.ApplicationGatewayProbePropertiesFormat.Port).Should(Equal(&annotationHpPort))
		})
		It("probe path must match annotation", func() {
			Expect(pb.ApplicationGatewayProbePropertiesFormat.Path).Should(Equal(&annotationHpPath))
		})
		It("probe status codes must match annotation", func() {
			Expect(pb.ApplicationGatewayProbePropertiesFormat.Match.StatusCodes).Should(Equal(&statusCodes))
		})
		It("probe interval must match annotation", func() {
			Expect(pb.ApplicationGatewayProbePropertiesFormat.Interval).Should(Equal(&annotationHpInterval))
		})
		It("probe timeout must match annotation", func() {
			Expect(pb.ApplicationGatewayProbePropertiesFormat.Timeout).Should(Equal(&annotationHpTimeout))
		})
		It("probe threshold must match annotation", func() {
			Expect(pb.ApplicationGatewayProbePropertiesFormat.UnhealthyThreshold).Should(Equal(&annotationHpThreshold))
		})
	})

})
