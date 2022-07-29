package azure

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure Auth Suite")
}

var _ = Describe("Azure", func() {
	Describe("Testing `azure auth` helpers", func() {

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
			It("getAuthorizer should try and get some authorizer but fail", func() {
				authorizer, err := getAuthorizer("", false, nil)
				Ω(err).To(HaveOccurred())
				Ω(authorizer).To(BeNil())
			})
			It("getAuthorizerWithRetry should try and get some authorizer but fail", func() {
				authorizer, err := GetAuthorizerWithRetry("", false, nil, 0, time.Duration(10))
				Ω(err).To(HaveOccurred())
				Ω(authorizer).To(BeNil())
			})
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
	})
})
