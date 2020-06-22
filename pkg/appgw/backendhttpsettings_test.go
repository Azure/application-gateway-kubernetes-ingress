// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
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
	endpoint := tests.NewEndpointsFixture()
	service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
	pod := tests.NewPodTestFixture(service.Namespace, "mybackend")
	ingress := tests.NewIngressFixture()
	_ = configBuilder.k8sContext.Caches.Pods.Add(&pod)
	_ = configBuilder.k8sContext.Caches.Endpoints.Add(endpoint)
	_ = configBuilder.k8sContext.Caches.Service.Add(service)
	_ = configBuilder.k8sContext.Caches.Ingress.Add(ingress)

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
})
