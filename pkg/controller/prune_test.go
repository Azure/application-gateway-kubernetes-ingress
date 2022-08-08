// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("prune function tests", func() {
	var controller *AppGwIngressController

	BeforeEach(func() {
		controller = &AppGwIngressController{
			appGwIdentifier: appgw.Identifier{
				SubscriptionID: "xxxx",
				ResourceGroup:  "xxxx",
				AppGwName:      "appgw",
			},
			recorder: record.NewFakeRecorder(100),
		}
	})

	Context("ensure pruneNoPrivateIP prunes ingress", func() {
		ingressPrivate := tests.NewIngressFixture()
		ingressPrivate.Annotations = map[string]string{
			annotations.UsePrivateIPKey: "true",
		}
		ingressPublic := tests.NewIngressFixture()
		cbCtx := &appgw.ConfigBuilderContext{
			IngressList: []*networking.Ingress{
				ingressPrivate,
				ingressPublic,
			},
			ServiceList: []*v1.Service{
				tests.NewServiceFixture(),
			},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}
		appGw := fixtures.GetAppGateway()

		It("removes the ingress using private ipAddress and keeps others", func() {
			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoPrivateIP(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(1))
			Expect(prunedIngresses[0].Annotations[annotations.UsePrivateIPKey]).To(Equal(""))
		})

		It("keeps the ingress using private ipAddress when public ipAddress is present", func() {
			appGw.FrontendIPConfigurations = &[]n.ApplicationGatewayFrontendIPConfiguration{
				fixtures.GetPublicIPConfiguration(),
				fixtures.GetPrivateIPConfiguration(),
			}
			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoPrivateIP(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(2))
		})
	})

	Context("ensure pruneNoSslCertificate prunes ingress", func() {
		ingressSslCertAnnotated := tests.NewIngressFixture()
		ingressSslCertAnnotated.Annotations = map[string]string{
			annotations.AppGwSslCertificate: "appgw-installed-cert",
		}
		ingressNoSslCertAnnotated := tests.NewIngressFixture()
		cbCtx := &appgw.ConfigBuilderContext{
			IngressList: []*networking.Ingress{
				ingressSslCertAnnotated,
				ingressNoSslCertAnnotated,
			},
			ServiceList: []*v1.Service{
				tests.NewServiceFixture(),
			},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}
		appGw := fixtures.GetAppGateway()

		It("removes the ingress using appgw-ssl-certificate and keeps others", func() {
			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoSslCertificate(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(1))
		})

		It("keeps the ingress using appgw-ssl-certificate when annotated ssl cert is pre-installed", func() {
			ingressSslCertAnnotated.Annotations = map[string]string{
				annotations.AppGwSslCertificate: *fixtures.GetCertificate1().Name,
			}

			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoSslCertificate(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(2))
		})
	})

	Context("ensure pruneNoSslProfile prunes ingress", func() {
		ingressSslProfileAnnotated := tests.NewIngressFixture()
		ingressSslProfileAnnotated.Annotations = map[string]string{
			annotations.AppGwSslProfile: "legacy-tls",
		}
		ingressNoSslProfileAnnotated := tests.NewIngressFixture()
		cbCtx := &appgw.ConfigBuilderContext{
			IngressList: []*networking.Ingress{
				ingressSslProfileAnnotated,
				ingressNoSslProfileAnnotated,
			},
			ServiceList: []*v1.Service{
				tests.NewServiceFixture(),
			},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}
		appGw := fixtures.GetAppGateway()

		It("removes the ingress using appgw-ssl-profile and keeps others", func() {
			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoSslProfile(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(1))
		})

		It("keeps the ingress using appgw-ssl-certificate when annotated ssl profile is pre-installed", func() {
			ingressSslProfileAnnotated.Annotations = map[string]string{
				annotations.AppGwSslProfile: *fixtures.GetSslProfile1().Name,
			}

			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoSslCertificate(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(2))
		})
	})

	Context("ensure pruneNoTrustedRootCertificate prunes ingress", func() {
		ingressRootCertAnnotated := tests.NewIngressFixture()
		ingressRootCertAnnotated.Annotations = map[string]string{
			annotations.AppGwTrustedRootCertificate: "appgw-installed-root-cert",
		}
		ingressNoRootCertAnnotated := tests.NewIngressFixture()
		cbCtx := &appgw.ConfigBuilderContext{
			IngressList: []*networking.Ingress{
				ingressRootCertAnnotated,
				ingressNoRootCertAnnotated,
			},
			ServiceList: []*v1.Service{
				tests.NewServiceFixture(),
			},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}
		appGw := fixtures.GetAppGateway()

		It("removes the ingress using appgw-trusted-root-certificate and keeps others", func() {
			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoTrustedRootCertificate(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(1))
		})

		It("keeps the ingress using appgw-trusted-root-certificate when annotated root cert is pre-installed", func() {
			// annotate with a installed root certificate
			installedRootCerts := fmt.Sprintf("%s,%s,%s", *fixtures.GetRootCertificate1().Name, *fixtures.GetRootCertificate2().Name, *fixtures.GetRootCertificate3().Name)
			ingressRootCertAnnotated.Annotations = map[string]string{
				annotations.AppGwTrustedRootCertificate: installedRootCerts,
			}

			Expect(len(cbCtx.IngressList)).To(Equal(2))
			prunedIngresses := pruneNoTrustedRootCertificate(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(prunedIngresses)).To(Equal(2))
		})
	})

	Context("ensure pruneRedirectNoTLS prunes ingress", func() {
		// invalid ingress without https and redirect
		ingressInvalid := tests.NewIngressFixture()
		ingressInvalid.Annotations = map[string]string{
			annotations.SslRedirectKey: "true",
		}
		ingressInvalid.Spec.TLS = nil
		It("should have ingressInvalid without https and redirect", func() {
			Expect(annotations.IsSslRedirect(ingressInvalid)).To(BeTrue())
			Expect(ingressInvalid.Spec.TLS).To(BeNil())
		})

		// valid ingress with https and redirect
		ingressValid1 := tests.NewIngressFixture()
		ingressValid1.Annotations = map[string]string{
			annotations.SslRedirectKey: "true",
		}
		It("should have ingressValid1 without https and redirect", func() {
			Expect(annotations.IsSslRedirect(ingressValid1)).To(BeTrue())
			Expect(ingressValid1.Spec.TLS).To(Not(BeNil()))
		})

		// valid ingress without https and redirect
		ingressValid2 := tests.NewIngressFixture()
		ingressValid2.Annotations = map[string]string{
			annotations.SslRedirectKey: "false",
		}
		ingressValid2.Spec.TLS = nil
		It("should have ingressValid2 without https and redirect", func() {
			Expect(annotations.IsSslRedirect(ingressValid2)).To(BeFalse())
			Expect(ingressValid2.Spec.TLS).To(BeNil())
		})

		cbCtx := &appgw.ConfigBuilderContext{
			IngressList: []*networking.Ingress{
				ingressInvalid,
				ingressValid1,
				ingressValid2,
			},
			ServiceList: []*v1.Service{
				tests.NewServiceFixture(),
			},
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
		}
		appGw := fixtures.GetAppGateway()
		It("removes the invalid ingresses", func() {
			prunedIngresses := pruneRedirectWithNoTLS(controller, &appGw, cbCtx, cbCtx.IngressList)
			Expect(len(cbCtx.IngressList)).To(Equal(3))
			Expect(len(prunedIngresses)).To(Equal(2))
			Expect(prunedIngresses).To(Not(ContainElement(ingressInvalid)))
			Expect(prunedIngresses).To(ContainElement(ingressValid1))
			Expect(prunedIngresses).To(ContainElement(ingressValid2))
		})
	})

	Context("ensure pruneProhibitedIngress prunes ingress", func() {
		env := environment.GetFakeEnv()
		env.EnableBrownfieldDeployment = true
		ingress := fixtures.GetIngressWithProhibitedTargetConflict()
		cbCtx := &appgw.ConfigBuilderContext{
			IngressList: []*networking.Ingress{
				ingress,
			},
			ServiceList: []*v1.Service{
				tests.NewServiceFixture(),
			},
			ProhibitedTargets:     fixtures.GetAzureIngressProhibitedTargets(),
			DefaultAddressPoolID:  to.StringPtr("xx"),
			DefaultHTTPSettingsID: to.StringPtr("yy"),
			EnvVariables:          env,
		}
		appGw := fixtures.GetAppGateway()

		validateOldIngress := func(oldIngress *networking.Ingress) {
			// should have two rules
			Expect(len(oldIngress.Spec.Rules)).To(Equal(2))

			// should have rule 1 with OldHost as host and no paths
			Expect(oldIngress.Spec.Rules[0].Host).To(Equal(tests.OtherHost))
			Expect(len(oldIngress.Spec.Rules[0].HTTP.Paths)).To(Equal(0))

			// should have rule 2 with Host as host and 2 path rules: /foo /fox
			Expect(oldIngress.Spec.Rules[1].Host).To(Equal(tests.Host))
			Expect(len(oldIngress.Spec.Rules[1].HTTP.Paths)).To(Equal(2))
			Expect(oldIngress.Spec.Rules[1].HTTP.Paths[0].Path).To(Equal(fixtures.PathFoo))
			Expect(oldIngress.Spec.Rules[1].HTTP.Paths[1].Path).To(Equal(fixtures.PathFox))
		}

		It("removes the ingress rules without modifying the original ingress", func() {
			Expect(len(cbCtx.IngressList)).To(Equal(1))

			// Get pointer to the old ingress object
			oldIngress := cbCtx.IngressList[0]

			// Validate that ingress follows the requirement
			validateOldIngress(oldIngress)

			// Prune: test.OtherHost and /fox are prohibited
			_ = pruneProhibitedIngress(controller, &appGw, cbCtx, cbCtx.IngressList)

			// Validate old ingress is the same as before
			validateOldIngress(oldIngress)

			// Validate new ingress
			newIngress := cbCtx.IngressList[0]

			Expect(len(newIngress.Spec.Rules)).To(Equal(1), "should have only 1 rule after pruning")
			Expect(len(newIngress.Spec.Rules[0].HTTP.Paths)).To(Equal(1), "Rule should have only 1 path rule left")
			Expect(oldIngress.Spec.Rules[1].Host).To(Equal(tests.Host), "Host for that path should be tests.Host")
			Expect(oldIngress.Spec.Rules[1].HTTP.Paths[0].Path).To(Equal(fixtures.PathFoo), "Path should /foo; /fox should be removed")
		})
	})
})
