// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cleanup", func() {
	Context("CleanUpPathRules", func() {
		var c *appGwConfigBuilder
		agicAddedPathRule := generatePathRuleName("test", "test", 0, 0)
		userAddedPathRule := "user-added-path-rule"

		BeforeEach(func() {
			c = &appGwConfigBuilder{
				appGw: n.ApplicationGateway{
					ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
						URLPathMaps: &[]n.ApplicationGatewayURLPathMap{
							{
								ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
									PathRules: &[]n.ApplicationGatewayPathRule{
										{
											Name: &agicAddedPathRule,
										},
										{
											Name: &userAddedPathRule,
										},
									},
								},
							},
						},
					},
				},
			}
		})

		It("should remove path rules that are created by AGIC", func() {
			c.CleanUpPathRulesAddedByAGIC()

			Expect(*c.appGw.URLPathMaps).To(HaveLen(1))

			pathRule := *(*c.appGw.URLPathMaps)[0].PathRules
			Expect(pathRule).To(HaveLen(1))
			Expect(*pathRule[0].Name).To(Equal(userAddedPathRule))
		})
	})
})
