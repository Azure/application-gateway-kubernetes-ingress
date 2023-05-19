// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build unittest

package azure

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Azure", func() {
	Describe("Testing `azure` helpers", func() {

		Context("ensure ParseResourceID works as expected", func() {
			It("should parse appgw resourceId correctly", func() {
				subID := SubscriptionID("xxxx")
				resGp := ResourceGroup("yyyy")
				resName := ResourceName("zzzz")
				resourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s", subID, resGp, resName)
				outSubID, outResGp, outResName := ParseResourceID(resourceID)
				Expect(outSubID).To(Equal(subID))
				Expect(resGp).To(Equal(outResGp))
				Expect(resName).To(Equal(outResName))
			})
		})

		Context("ensure ConvertToClusterResourceGroup works as expected", func() {
			It("should parse empty infra resource group correctly", func() {
				subID := SubscriptionID("xxxx")
				resGp := ResourceGroup("")
				_, err := ConvertToClusterResourceGroup(subID, resGp, nil)
				Ω(err).To(HaveOccurred(), "this call should have failed in parsing the resource group")
			})
			It("should parse valid infra resource group correctly", func() {
				subID := SubscriptionID("xxxx")
				resGp := ResourceGroup("MC_resgp_resName_location")
				Expect(ConvertToClusterResourceGroup(subID, resGp, nil)).To(Equal("/subscriptions/xxxx/resourcegroups/resgp/providers/Microsoft.ContainerService/managedClusters/resName"))

				subID = SubscriptionID("xxxx")
				resGp = ResourceGroup("mc_resgp_resName_location")
				Expect(ConvertToClusterResourceGroup(subID, resGp, nil)).To(Equal("/subscriptions/xxxx/resourcegroups/resgp/providers/Microsoft.ContainerService/managedClusters/resName"))
			})
		})

		Context("test getAuthorizer functions", func() {
			BeforeEach(func() {
				os.Setenv(auth.ClientID, "guid1")
				os.Setenv(auth.TenantID, "guid2")
				os.Setenv(auth.ClientSecret, "fake-secret")
			})

			It("getAuthorizer should try and get some authorizer", func() {
				authorizer, err := getAuthorizer("", false, nil)
				Ω(err).ToNot(HaveOccurred())
				Ω(authorizer).ToNot(BeNil())
			})

			It("getAuthorizerWithRetry should try and get some authorizer", func() {
				authorizer, err := GetAuthorizerWithRetry("", false, nil, 0, time.Duration(10))
				Ω(err).ToNot(HaveOccurred())
				Ω(authorizer).ToNot(BeNil())
			})

			AfterEach(func() {
				os.Unsetenv(auth.ClientID)
				os.Unsetenv(auth.TenantID)
				os.Unsetenv(auth.ClientSecret)
			})
		})

		Context("test AzContext struct", func() {
			cpConfigFile := `{
				"cloud": "xxxx",
				"tenantId": "t",
				"subscriptionId": "s",
				"aadClientId": "c",
				"aadClientSecret": "cs",
				"resourceGroup": "r",
				"location": "l",
				"vmType": "xxxx",
				"subnetName": "xxxx",
				"securityGroupName": "xxxx",
				"vnetName": "xxxx",
				"vnetResourceGroup": "xxxx",
				"routeTableName": "xxxx",
				"primaryAvailabilitySetName": "xxxx",
				"primaryScaleSetName": "xxxx",
				"cloudProviderBackoff": "xxxx",
				"cloudProviderBackoffRetries": "xxxx",
				"cloudProviderBackoffExponent": "xxxx",
				"cloudProviderBackoffDuration": "xxxx",
				"cloudProviderBackoffJitter": "xxxx",
				"cloudProviderRatelimit": "xxxx",
				"cloudProviderRateLimitQPS": "xxxx",
				"cloudProviderRateLimitBucket": "xxxx",
				"useManagedIdentityExtension": "xxxx",
				"userAssignedIdentityID": "xxxx",
				"useInstanceMetadata": true,
				"loadBalancerSku": "xxxx",
				"excludeMasterFromStandardLB": "xxxx",
				"providerVaultName": "xxxx",
				"maximumLoadBalancerRuleCount": "xxxx",
				"providerKeyName": "xxxx",
				"providerKeyVersion": "xxxx"
			}`

			It("should deserialize correctly", func() {
				var cpConfig CloudProviderConfig
				err := json.Unmarshal([]byte(cpConfigFile), &cpConfig)
				Ω(err).ToNot(HaveOccurred())
				Ω(cpConfig.TenantID).To(Equal("t"))
				Ω(cpConfig.Region).To(Equal("l"))
			})
		})

		Context("test RouteTableID func", func() {
			It("generate correct route table ID", func() {
				expectedRouteTable := "/subscriptions/subID/resourceGroups/resGp/providers/Microsoft.Network/routeTables/rt"
				Expect(RouteTableID(SubscriptionID("subID"), ResourceGroup("resGp"), ResourceName("rt"))).To(Equal(expectedRouteTable))
			})
		})

		Context("test ApplicationGatewayID func", func() {
			It("generate correct application gateway ID", func() {
				expectedGatewayID := "/subscriptions/subID/resourceGroups/resGp/providers/Microsoft.Network/applicationGateways/gateway"
				Expect(ApplicationGatewayID(SubscriptionID("subID"), ResourceGroup("resGp"), ResourceName("gateway"))).To(Equal(expectedGatewayID))
			})
		})

		Context("test ResourceGroupID func", func() {
			It("generate correct resource group ID", func() {
				expectedGroupID := "/subscriptions/subID/resourceGroups/resGp"
				Expect(ResourceGroupID(SubscriptionID("subID"), ResourceGroup("resGp"))).To(Equal(expectedGroupID))
			})
		})

		Context("test ParseSubResourceID func", func() {
			It("parses sub resource ID correctly", func() {
				subResourceID := "/subscriptions/subID/resourceGroups/resGp/providers/Microsoft.Network/applicationGateways/appgw/sslCertificates/cert"
				subID, resourceGp, resource, subResource := ParseSubResourceID(subResourceID)
				Expect(subID).To(Equal(SubscriptionID("subID")))
				Expect(resourceGp).To(Equal(ResourceGroup("resGp")))
				Expect(resource).To(Equal(ResourceName("appgw")))
				Expect(subResource).To(Equal(ResourceName("cert")))
			})

			It("should give error if segements are less", func() {
				subResourceID := "/subscriptions/subID/resourceGroups/resGp/providers/Microsoft.Network/applicationGateways/appgw"
				subID, resourceGp, resource, subResource := ParseSubResourceID(subResourceID)
				Expect(subID).To(Equal(SubscriptionID("")))
				Expect(resourceGp).To(Equal(ResourceGroup("")))
				Expect(resource).To(Equal(ResourceName("")))
				Expect(subResource).To(Equal(ResourceName("")))
			})
		})

		Context("test GetOperationIDFromPollingURL func", func() {
			It("should be able to parse operationID", func() {
				pollingURL := "https://management.azure.com/subscriptions/87654321-abcd-1234-b193-d305572e416f/providers/Microsoft.Network/locations/eastus2/operations/c24da597-9666-4950-a9f7-10bdfa17883d?api-version=2020-05-01"
				expectedOperationID := "c24da597-9666-4950-a9f7-10bdfa17883d"
				Expect(GetOperationIDFromPollingURL(pollingURL)).To(Equal(expectedOperationID))
			})

			It("should give empty string when unable to parse", func() {
				pollingURL := "random"
				expectedOperationID := ""
				Expect(GetOperationIDFromPollingURL(pollingURL)).To(Equal(expectedOperationID))
			})
		})
	})
})
