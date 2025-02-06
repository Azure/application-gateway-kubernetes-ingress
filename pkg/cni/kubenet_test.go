package cni_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl_client "sigs.k8s.io/controller-runtime/pkg/client"
	ctrl_client_fake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/cni"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8s"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

var _ = Describe("Kubenet CNI", func() {
	var ctx context.Context
	var cancel context.CancelFunc
	var azClient *azure.FakeAzClient
	var k8sClient ctrl_client.Client
	var recorder *record.FakeRecorder
	var agicPod *v1.Pod
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
	var reconciler *cni.Reconciler

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		azClient = azure.NewFakeAzClient()
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		agicPod = &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "agic",
				Namespace: "kube-system",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
		}

		scheme, _ := k8s.NewScheme()
		k8sClient = ctrl_client_fake.NewClientBuilder().WithScheme(scheme).Build()

		reconciler = cni.NewReconciler(azClient, k8sClient, recorder, &azure.CloudProviderConfig{
			SubscriptionID:          "test-sub",
			RouteTableResourceGroup: "test-rg",
			RouteTableName:          "test-rt",
		}, appGw, agicPod, "test", false)
	})

	Context("reconcileKubenetCniIfNeeded", func() {
		It("should apply route table", func() {
			azClient.ApplyRouteTableFunc = func(subnetID string, routeTableID string) error {
				Expect(subnetID).To(Equal("subnet-id"))
				Expect(routeTableID).To(Equal("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/routeTables/test-rt"))
				return nil
			}

			reconciler.Reconcile(ctx)
			Eventually(recorder.Events).ShouldNot(Receive())
		})

		It("should return nil if RouteTableName is empty", func() {
			azClient.ApplyRouteTableFunc = func(subnetID string, routeTableID string) error {
				Fail("ApplyRouteTable should not be called")
				return nil
			}

			reconciler = cni.NewReconciler(azClient, k8sClient, recorder, &azure.CloudProviderConfig{
				SubscriptionID: "test-sub",
			}, appGw, agicPod, "test", false)

			err := reconciler.Reconcile(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create event if ApplyRouteTable fails", func() {
			azClient.ApplyRouteTableFunc = func(subnetID string, routeTableID string) error {
				return errors.New("failed to apply route table")
			}

			err := reconciler.Reconcile(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to apply route table"))
		})
	})

	AfterEach(func() {
		cancel()
	})
})
