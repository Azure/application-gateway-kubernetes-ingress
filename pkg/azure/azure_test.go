// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure Suite")
}

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
			It("should parse empty infra resourse group correctly", func() {
				subID := SubscriptionID("xxxx")
				resGp := ResourceGroup("")
				_, err := ConvertToClusterResourceGroup(subID, resGp, nil)
				Ω(err).To(HaveOccurred(), "this call should have failed in parsing the resource group")
			})
			It("should parse valid infra resourse group correctly", func() {
				subID := SubscriptionID("xxxx")
				resGp := ResourceGroup("MC_resgp_resName_location")
				Expect(ConvertToClusterResourceGroup(subID, resGp, nil)).To(Equal("/subscriptions/xxxx/resourcegroups/resgp/providers/Microsoft.ContainerService/managedClusters/resName"))

				subID = SubscriptionID("xxxx")
				resGp = ResourceGroup("mc_resgp_resName_location")
				Expect(ConvertToClusterResourceGroup(subID, resGp, nil)).To(Equal("/subscriptions/xxxx/resourcegroups/resgp/providers/Microsoft.ContainerService/managedClusters/resName"))
			})
		})

		Context("test getAuthorizer", func() {
			It("should try and get some authorizer", func() {
				authorizer, err := getAuthorizer("", false, nil)
				Ω(authorizer).ToNot(BeNil())
				Ω(err).ToNot(HaveOccurred())
			})
		})

		Context("test getAuthorizerWithRetry", func() {
			It("should try and get some authorizer", func() {
				authorizer, err := GetAuthorizerWithRetry("", false, nil, 0, time.Duration(10))
				Ω(authorizer).ToNot(BeNil())
				Ω(err).ToNot(HaveOccurred())
			})
		})

		Context("test waitForAzureAuth", func() {
			client := NewFakeAzClient()
			client.GetGatewayFunc = GetGatewayFunc(func() (n.ApplicationGateway, error) {
				return n.ApplicationGateway{}, errors.New("some error")
			})
			It("should try and panic", func() {
				err := WaitForAzureAuth(client, 0, time.Duration(10))
				Ω(err).To(HaveOccurred())
			})
		})

		Context("test AzContext struct", func() {
			contextFile := `{
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
				var context AzContext
				err := json.Unmarshal([]byte(contextFile), &context)
				Ω(err).ToNot(HaveOccurred())
				Ω(context.TenantID).To(Equal("t"))
				Ω(context.Region).To(Equal("l"))
			})
		})
	})
})
