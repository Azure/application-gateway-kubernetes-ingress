package cni_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl_client "sigs.k8s.io/controller-runtime/pkg/client"
	ctrl_client_fake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/cni"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8s"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

var _ = Describe("Kubenet CNI", func() {
	var ctx = context.TODO()
	var azClient *azure.FakeAzClient
	var k8sClient ctrl_client.Client
	var appGw = n.ApplicationGateway{
		ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
			GatewayIPConfigurations: &[]n.ApplicationGatewayIPConfiguration{
				{
					ApplicationGatewayIPConfigurationPropertiesFormat: &n.ApplicationGatewayIPConfigurationPropertiesFormat{
						Subnet: &n.SubResource{
							ID: to.StringPtr("subnet-id"),
						},
					},
				},
			},
		},
	}

	BeforeEach(func() {
		azClient = azure.NewFakeAzClient()

		scheme, _ := k8s.NewScheme()
		k8sClient = ctrl_client_fake.NewClientBuilder().WithScheme(scheme).Build()
	})

	Context("reconcileKubenetCniIfNeeded", func() {
		It("should apply route table", func() {
			azClient.ApplyRouteTableFunc = func(subnetID string, routeTableID string) error {
				Expect(subnetID).To(Equal("subnet-id"))
				Expect(routeTableID).To(Equal("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/routeTables/test-rt"))
				return nil
			}

			err := cni.ReconcileCNI(ctx, azClient, k8sClient, "test", &azure.CloudProviderConfig{
				SubscriptionID:          "test-sub",
				RouteTableResourceGroup: "test-rg",
				RouteTableName:          "test-rt",
			}, appGw, false)
			Expect(err).To(BeNil())
		})

		It("should return nil if RouteTableName is empty", func() {
			azClient.ApplyRouteTableFunc = func(subnetID string, routeTableID string) error {
				Fail("ApplyRouteTable should not be called")
				return nil
			}

			err := cni.ReconcileCNI(ctx, azClient, k8sClient, "test", &azure.CloudProviderConfig{
				RouteTableName: "",
			}, appGw, false)
			Expect(err).To(BeNil())
		})
	})
})
