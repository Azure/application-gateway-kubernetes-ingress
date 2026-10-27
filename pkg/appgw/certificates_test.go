package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	networking "k8s.io/api/networking/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("Testing function newHostToSecretMap", func() {
	const host1 = "ftp.contoso.com"
	const host2 = "www.contoso.com"
	expectedHostToSecretMap := map[string]secretIdentifier{
		host1: {
			tests.Namespace,
			tests.NameOfSecret,
		},
		host2: {
			tests.Namespace,
			tests.NameOfSecret,
		},
		tests.Host: {
			tests.Namespace,
			tests.NameOfSecret,
		},
		"": {
			tests.Namespace,
			tests.NameOfSecret,
		},
	}

	expectedSecret := secretIdentifier{
		Namespace: tests.Namespace,
		Name:      tests.NameOfSecret,
	}

	Context("Test fetching secrets from ingress with TLS spec", func() {
		cb := newConfigBuilderFixture(nil)
		ingress := tests.NewIngressFixture()

		actualHostToSecretMap := cb.newHostToSecretMap(ingress)

		It("should have generated the expected host to secret map", func() {
			Expect(actualHostToSecretMap).To(Equal(expectedHostToSecretMap))
		})
		It("should have correct keys", func() {
			var keys []string
			for k := range actualHostToSecretMap {
				keys = append(keys, k)
			}

			// We check each key to ensure that unstable sort does not cause test flakiness
			Expect(keys).To(ContainElement(tests.Host))
			Expect(keys).To(ContainElement(host1))
			Expect(keys).To(ContainElement(host2))
			Expect(keys).To(ContainElement(""))
		})

		It("has the correct secrets", func() {
			Expect(actualHostToSecretMap[tests.Host]).To(Equal(expectedSecret))
		})
	})

	Context("Test obtaining a single certificate for an existing host", func() {
		cb := newConfigBuilderFixture(nil)
		ingress := tests.NewIngressFixture()
		hostnameSecretIDMap := cb.newHostToSecretMap(ingress)
		actualSecret, actualSecretID := cb.getCertificate(ingress, host1, hostnameSecretIDMap)

		It("should have generated the expected secret", func() {
			Expect(*actualSecret).To(Equal("eHl6"))
		})

		It("should have generated the correct secretID struct", func() {
			Expect(*actualSecretID).To(Equal(expectedSecret))
		})
	})
})

var _ = Describe("SSL certificate garbage collection", func() {
	It("removes stale AGIC-managed certs but retains manual certs", func() {
		cb := newConfigBuilderFixture(nil)

		staleManagedName := secretIdentifier{Namespace: tests.Namespace, Name: "stale-secret"}.secretFullName()
		manualName := "manual-cert"
		cb.appGw.SslCertificates = &[]n.ApplicationGatewaySslCertificate{
			{Name: to.StringPtr(staleManagedName)},
			{Name: to.StringPtr(manualName)},
		}

		ingressList := []*networking.Ingress{tests.NewIngressFixture()}
		cbCtx := &ConfigBuilderContext{
			IngressList:  ingressList,
			EnvVariables: environment.EnvVariables{},
		}

		actual := cb.getSslCertificates(cbCtx)
		var names []string
		for _, cert := range *actual {
			names = append(names, *cert.Name)
		}

		expectedDesiredName := secretIdentifier{Namespace: tests.Namespace, Name: tests.NameOfSecret}.secretFullName()
		Expect(names).To(ContainElement(expectedDesiredName))
		Expect(names).To(ContainElement(manualName))
		Expect(names).NotTo(ContainElement(staleManagedName))
	})

	It("retains AGIC-managed certs referenced by blacklisted listeners in brownfield mode", func() {
		cb := newConfigBuilderFixture(nil)

		staleManagedName := secretIdentifier{Namespace: tests.Namespace, Name: "stale-secret"}.secretFullName()
		cb.appGw.SslCertificates = &[]n.ApplicationGatewaySslCertificate{
			{Name: to.StringPtr(staleManagedName)},
		}

		listenerName := "bf-listener"
		cb.appGw.HTTPListeners = &[]n.ApplicationGatewayHTTPListener{
			{
				Name: to.StringPtr(listenerName),
				ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
					Protocol:       n.ApplicationGatewayProtocolHTTPS,
					HostName:       to.StringPtr(tests.Host),
					SslCertificate: &n.SubResource{ID: to.StringPtr(cb.appGwIdentifier.sslCertificateID(staleManagedName))},
				},
			},
		}

		cbCtx := &ConfigBuilderContext{
			IngressList:       []*networking.Ingress{tests.NewIngressFixture()},
			ProhibitedTargets: fixtures.GetAzureIngressProhibitedTargets(),
			EnvVariables: environment.EnvVariables{
				EnableBrownfieldDeployment: true,
			},
		}

		actual := cb.getSslCertificates(cbCtx)
		var names []string
		for _, cert := range *actual {
			names = append(names, *cert.Name)
		}
		Expect(names).To(ContainElement(staleManagedName))
	})

	It("retains certificates referenced by appgw-ssl-certificate annotation", func() {
		cb := newConfigBuilderFixture(nil)

		// This resembles an AGIC-managed name when APPGW_CONFIG_NAME_PREFIX is empty.
		annotatedCertName := "cert-some-namespace-some-secret"
		cb.appGw.SslCertificates = &[]n.ApplicationGatewaySslCertificate{
			{Name: to.StringPtr(annotatedCertName)},
		}

		ingress := tests.NewIngressFixture()
		ingress.Spec.TLS = nil
		if ingress.Annotations == nil {
			ingress.Annotations = map[string]string{}
		}
		ingress.Annotations[annotations.AppGwSslCertificate] = annotatedCertName

		cbCtx := &ConfigBuilderContext{
			IngressList:  []*networking.Ingress{ingress},
			EnvVariables: environment.EnvVariables{},
		}

		actual := cb.getSslCertificates(cbCtx)
		var names []string
		for _, cert := range *actual {
			names = append(names, *cert.Name)
		}
		Expect(names).To(ContainElement(annotatedCertName))
	})
})
