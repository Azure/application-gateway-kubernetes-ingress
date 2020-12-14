// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"strconv"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"

	"strings"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Test the creation of Backend http settings from Ingress definition", func() {
	// Setup
	configBuilder := newConfigBuilderFixture(nil)

	// contains endpoint for the service ports. Multiple ports present with name as https-port// contains endpoint for the service ports. Multiple ports present with name as https-port
	endpoint := tests.NewEndpointsFixtureWithSameNameMultiplePorts()

	// service "--service-name--" contains multiple service ports with port as 80, 443, etc and target port as 9876, pod port name
	service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)

	pod := tests.NewPodTestFixture(service.Namespace, "mybackend")

	// Ingress "--name--" contains two rules with service port as 80 and 443
	ingress := tests.NewIngressFixture()
	_ = configBuilder.k8sContext.Caches.Pods.Add(&pod)
	_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
	_ = configBuilder.k8sContext.Caches.Service.Add(service)
	_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress)

	// Ingress "websocket-ingress" with service missing "websocket-service"
	ingress1, _ := tests.GetVerySimpleIngress()
	_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress1)

	Context("test backend protocol annotation configures protocol on httpsettings and probes when no readiness probe on the pods", func() {

		// checkBackendProtocolAnnotation function calls generates backend http settings map
		// based on backend protocol annotation and then test against expected backend http settings.
		checkBackendProtocolAnnotation := func(annotationValue string, protocolEnum annotations.ProtocolEnum, expectedProtocolValue n.ApplicationGatewayProtocol) {
			// Setup
			ingress.Annotations[annotations.BackendProtocolKey] = annotationValue
			_ = configBuilder.k8sContext.Caches.Ingress.Update(ingress)
			Expect(annotations.BackendProtocol(ingress)).To(Equal(protocolEnum))

			cbCtx := &ConfigBuilderContext{
				IngressList:           []*v1beta1.Ingress{ingress},
				ServiceList:           []*v1.Service{service},
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			// Action
			configBuilder.mem = memoization{}
			probes, _ := configBuilder.newProbesMap(cbCtx)
			httpSettings, _, _, _ := configBuilder.getBackendsAndSettingsMap(cbCtx)

			for _, setting := range httpSettings {
				if *setting.Name == DefaultBackendHTTPSettingsName {
					Expect(setting.Protocol).To(Equal(n.HTTP), "default backend %s should have %s", *setting.Name, n.HTTP)
					Expect(probes[utils.GetLastChunkOfSlashed(*setting.Probe.ID)].Protocol).To(Equal(n.HTTP), "default probe should have http")
					continue
				}

				Expect(setting.Protocol).To(Equal(expectedProtocolValue), "backend %s should have %s", *setting.Name, expectedProtocolValue)
				Expect(probes[utils.GetLastChunkOfSlashed(*setting.Probe.ID)].Protocol).To(Equal(expectedProtocolValue), "probe should have same protocol as http setting")
			}
		}

		It("should have all but default backend http settings with https", func() {
			checkBackendProtocolAnnotation("HttPS", annotations.HTTPS, n.HTTPS)
		})

		It("should have all backend http settings with http", func() {
			checkBackendProtocolAnnotation("HttP", annotations.HTTP, n.HTTP)
		})
	})

	Context("test appgw trusted root certificate annotation configures trusted root certificate(s) on httpsettings", func() {

		checkTrustedRootCertificateAnnotation := func(protocol string, trustedRootCertificate string, protocolEnum annotations.ProtocolEnum, expectedProtocolValue n.ApplicationGatewayProtocol) {
			// appgw trusted root certificate needs to be used together with backend protocal annotation, and protocal "https" should be used.
			// PickHostNameFromBackendAddress will be true given backend hostname is not specified
			ingress.Annotations[annotations.BackendProtocolKey] = protocol
			ingress.Annotations[annotations.AppGwTrustedRootCertificate] = trustedRootCertificate
			_ = configBuilder.k8sContext.Caches.Ingress.Update(ingress)

			cbCtx := &ConfigBuilderContext{
				IngressList:           []*v1beta1.Ingress{ingress},
				ServiceList:           []*v1.Service{service},
				DefaultAddressPoolID:  to.StringPtr("xx"),
				DefaultHTTPSettingsID: to.StringPtr("yy"),
			}

			// Action
			configBuilder.mem = memoization{}
			probes, _ := configBuilder.newProbesMap(cbCtx)
			httpSettings, _, _, _ := configBuilder.getBackendsAndSettingsMap(cbCtx)

			for _, setting := range httpSettings {
				if *setting.Name == DefaultBackendHTTPSettingsName {
					Expect(setting.Protocol).To(Equal(n.HTTP), "default backend %s should have %s", *setting.Name, n.HTTP)
					Expect(probes[utils.GetLastChunkOfSlashed(*setting.Probe.ID)].Protocol).To(Equal(n.HTTP), "default probe should have http")
					continue
				}

				Expect(setting.Protocol).To(Equal(expectedProtocolValue), "backend %s should have %s", *setting.Name, expectedProtocolValue)
				Expect(probes[utils.GetLastChunkOfSlashed(*setting.Probe.ID)].Protocol).To(Equal(expectedProtocolValue), "probe should have same protocol as http setting")
				Expect(len(*setting.TrustedRootCertificates)).To(Equal(2), "backend %s should have one two trusted root certificates configured", *setting.Name)
				for _, certID := range *setting.TrustedRootCertificates {
					segments := strings.Split(*certID.ID, "/")
					certName := segments[len(segments)-1]
					Expect(strings.Contains("rootcert1,rootcert2", certName)).To(Equal(true), "root certificate %s is not found", certName)
				}
			}
		}

		It("should have all but default backend http settings with https and trusted root certificates", func() {
			checkTrustedRootCertificateAnnotation("Https", "rootcert1,rootcert2", annotations.HTTPS, n.HTTPS)
		})

	})

	Context("test backend ports for the http settings", func() {
		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		configBuilder.mem = memoization{}
		configBuilder.newProbesMap(cbCtx)
		httpSettings, _, _, _ := configBuilder.getBackendsAndSettingsMap(cbCtx)

		It("correct backend port is choosen in case of target port is resolved to multiple ports", func() {
			expectedhttpSettingsLen := 3
			Expect(expectedhttpSettingsLen).To(Equal(len(httpSettings)), "httpSetting count %d should be %d", len(httpSettings), expectedhttpSettingsLen)

			for _, setting := range httpSettings {
				if *setting.Name == DefaultBackendHTTPSettingsName {
					Expect(int32(80)).To(Equal(*setting.Port), "default backend port %d should be 80", *setting.Port)
				} else if strings.Contains(*setting.Name, strconv.Itoa(int(tests.ContainerPort))) {
					// http setting for ingress with service port as 80
					Expect(tests.ContainerPort).To(Equal(*setting.Port), "setting %s backend port %d should be 9876", *setting.Name, *setting.Port)
				} else if strings.Contains(*setting.Name, "75") {
					// http setting for the ingress with service port as 443. Target port is https-port which resolves to multiple backend port
					// and the smallest backend port is chosen
					Expect(int32(75)).To(Equal(*setting.Port), "setting %s backend port %d should be 75", *setting.Name, *setting.Port)
				} else {
					// Dummy Failure, This should not happen
					Expect(23).To(Equal(75), "setting %s is not expected to be created", *setting.Name)
				}
			}
		})
	})

	Context("make sure all backends are processed", func() {
		cbCtx := &ConfigBuilderContext{
			IngressList:           []*v1beta1.Ingress{ingress, ingress1},
			ServiceList:           []*v1.Service{service},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}

		configBuilder.mem = memoization{}
		configBuilder.newProbesMap(cbCtx)
		httpSettings, _, _, _ := configBuilder.getBackendsAndSettingsMap(cbCtx)

		It("should configure all the backends even when a service is missing", func() {
			expectedhttpSettingsLen := 4
			Expect(expectedhttpSettingsLen).To(Equal(len(httpSettings)), "httpSetting count %d should be %d", len(httpSettings), expectedhttpSettingsLen)

			for _, setting := range httpSettings {
				if *setting.Name == DefaultBackendHTTPSettingsName {
					Expect(int32(80)).To(Equal(*setting.Port), "default backend port %d should be 80", *setting.Port)
				} else if strings.Contains(*setting.Name, strconv.Itoa(int(tests.ContainerPort))) {
					// http setting for ingress with service port as 80
					Expect(tests.ContainerPort).To(Equal(*setting.Port), "setting %s backend port %d should be 9876", *setting.Name, *setting.Port)
				} else if strings.Contains(*setting.Name, "75") {
					// http setting for the ingress with service port as 443. Target port is https-port which resolves to multiple backend port
					// and the smallest backend port is chosen
					Expect(int32(75)).To(Equal(*setting.Port), "setting %s backend port %d should be 75", *setting.Name, *setting.Port)
				} else if strings.Contains(*setting.Name, "bp--websocket-service-80-80-websocket-ingress") {
					// http setting for the ingress with service port as 443. Target port is https-port which resolves to multiple backend port
					// and the smallest backend port is chosen
					Expect(int32(80)).To(Equal(*setting.Port), "setting %s backend port %d should be 75", *setting.Name, *setting.Port)
				} else {
					// Dummy Failure, This should not happen
					Expect(23).To(Equal(75), "setting %s is not expected to be created", *setting.Name)
				}
			}
		})
	})
})
