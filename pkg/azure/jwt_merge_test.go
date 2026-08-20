// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MergeEntraJWTConfigs", func() {
	It("preserves existing configs not in helm list", func() {
		existing := []EntraJWTValidationConfig{
			{Name: to.StringPtr("portal-jwt"), Properties: &EntraJWTValidationConfigPropertiesFormat{TenantID: to.StringPtr("t1"), ClientID: to.StringPtr("c1")}},
		}
		merged := MergeEntraJWTConfigs(existing, nil)
		Expect(merged).To(HaveLen(1))
		Expect(*merged[0].Name).To(Equal("portal-jwt"))
	})

	It("upserts helm config over same name", func() {
		existing := []EntraJWTValidationConfig{
			{Name: to.StringPtr("ims-jwt"), Properties: &EntraJWTValidationConfigPropertiesFormat{TenantID: to.StringPtr("old"), ClientID: to.StringPtr("old")}},
		}
		helm := []EntraJWTValidationConfig{
			NewHelmEntraJWTConfig("sub", "rg", "gw", "ims-jwt", "new-tenant", "new-client", "Deny", []string{"api://x"}),
		}
		merged := MergeEntraJWTConfigs(existing, helm)
		Expect(merged).To(HaveLen(1))
		Expect(*merged[0].Properties.TenantID).To(Equal("new-tenant"))
		Expect(*merged[0].Properties.ClientID).To(Equal("new-client"))
		Expect(*merged[0].Properties.Audiences).To(Equal([]string{"api://x"}))
	})

	It("adds helm configs that do not exist yet and keeps others", func() {
		existing := []EntraJWTValidationConfig{
			{Name: to.StringPtr("portal-jwt"), Properties: &EntraJWTValidationConfigPropertiesFormat{TenantID: to.StringPtr("t1"), ClientID: to.StringPtr("c1")}},
		}
		helm := []EntraJWTValidationConfig{
			NewHelmEntraJWTConfig("sub", "rg", "gw", "ims-jwt", "t2", "c2", "Allow", nil),
		}
		merged := MergeEntraJWTConfigs(existing, helm)
		Expect(merged).To(HaveLen(2))
		names := ConfigNames(merged)
		Expect(names).To(HaveKey("portal-jwt"))
		Expect(names).To(HaveKey("ims-jwt"))
	})
})
