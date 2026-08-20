// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

//go:build unittest
// +build unittest

package appgw

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	networking "k8s.io/api/networking/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/go-autorest/autorest/to"
)

var _ = Describe("Entra JWT merge payload", func() {
	Context("Helm upsert + preserve", func() {
		It("builds payload with helm upsert and preserved portal config", func() {
			cb := newConfigBuilderFixture(nil)
			cbCtx := &ConfigBuilderContext{
				EnvVariables: environment.EnvVariables{
					EntraJWTConfigs: []environment.EntraJWTConfig{
						{Name: "ims-jwt", TenantID: "tenant", ClientID: "client", UnauthorizedAction: "Deny"},
					},
				},
				ExistingEntraJWTConfigs: []azure.EntraJWTValidationConfig{
					{Name: to.StringPtr("portal-jwt"), Properties: &azure.EntraJWTValidationConfigPropertiesFormat{
						TenantID: to.StringPtr("pt"), ClientID: to.StringPtr("pc"),
					}},
				},
			}

			payload := cb.buildEntraJWTMergePayload(cbCtx, map[string]string{"rr-1": "ims-jwt"})
			Expect(payload.Configs).To(HaveLen(2))
			Expect(payload.RuleBindings).To(HaveKeyWithValue("rr-1", "ims-jwt"))
			names := azure.ConfigNames(payload.Configs)
			Expect(names).To(HaveKey("ims-jwt"))
			Expect(names).To(HaveKey("portal-jwt"))
		})
	})

	Context("ingress never creates JWT configs", func() {
		It("does not create JWT configs from ingress annotations alone", func() {
			cb := newConfigBuilderFixture(nil)
			ing := tests.NewIngressFixture()
			ing.Annotations = map[string]string{
				annotations.IngressClassKey:       tests.IngressClassController,
				annotations.EntraJWTConfigNameKey: "ghost-jwt",
			}
			cbCtx := &ConfigBuilderContext{
				IngressList:  []*networking.Ingress{ing},
				EnvVariables: environment.GetFakeEnv(),
			}
			payload := cb.buildEntraJWTMergePayload(cbCtx, nil)
			Expect(payload.Configs).To(BeEmpty())
		})
	})
})
