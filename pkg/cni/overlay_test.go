package cni_test

import (
	"context"
	"errors"
	"time"

	nodenetworkconfig_v1alpha "github.com/Azure/azure-container-networking/crd/nodenetworkconfig/api/v1alpha"
	overlayextensionconfig_v1alpha1 "github.com/Azure/azure-container-networking/crd/overlayextensionconfig/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("Overlay CNI", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		azClient   *azure.FakeAzClient
		k8sClient  ctrl_client.Client
		recorder   *record.FakeRecorder
		agicPod    *v1.Pod
		namespace  = "test-namespace"
		subnetCIDR = "10.0.0.0/16"
		appGw      n.ApplicationGateway
		reconciler *cni.Reconciler
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		azClient = azure.NewFakeAzClient()
		azClient.ApplyRouteTableFunc = func(subnetID, routeTableID string) error {
			Fail("ApplyRouteTable should not be called")
			return nil
		}

		scheme, _ := k8s.NewScheme()
		k8sClient = ctrl_client_fake.NewClientBuilder().WithScheme(scheme).Build()
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true
		agicPod = &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "agic", Namespace: "kube-system"},
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		}

		appGw = n.ApplicationGateway{
			ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
				GatewayIPConfigurations: &[]n.ApplicationGatewayIPConfiguration{
					{
						ApplicationGatewayIPConfigurationPropertiesFormat: &n.ApplicationGatewayIPConfigurationPropertiesFormat{
							Subnet: &n.SubResource{ID: to.StringPtr("subnet-id")},
						},
					},
				},
			},
		}

		reconciler = cni.NewReconciler(azClient, k8sClient, recorder, &azure.CloudProviderConfig{
			SubscriptionID: "test-sub",
		}, appGw, agicPod, namespace, false)
	})

	Context("Handle Overlay CNI cluster", func() {
		BeforeEach(func() {
			// Create NodeNetworkConfig so that cluster is considered as overlay CNI
			config := &nodenetworkconfig_v1alpha.NodeNetworkConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node-network-config", Namespace: namespace},
				Spec:       nodenetworkconfig_v1alpha.NodeNetworkConfigSpec{},
			}
			Expect(k8sClient.Create(ctx, config)).To(BeNil())

			// Run a goroutine to update the status of
			// OverlayExtensionConfig to Succeeded.
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-time.After(1 * time.Second):
						var config overlayextensionconfig_v1alpha1.OverlayExtensionConfig
						if err := k8sClient.Get(ctx, ctrl_client.ObjectKey{Name: cni.OverlayExtensionConfigName, Namespace: namespace}, &config); err != nil {
							continue
						}

						config.Status.State = overlayextensionconfig_v1alpha1.Succeeded
						_ = k8sClient.Update(ctx, &config)
						return
					}
				}
			}()
		})

		When("OEC doesn't exist", func() {
			BeforeEach(func() {
				azClient.GetSubnetFunc = func(subnetID string) (n.Subnet, error) {
					return n.Subnet{SubnetPropertiesFormat: &n.SubnetPropertiesFormat{AddressPrefix: to.StringPtr(subnetCIDR)}}, nil
				}
			})

			It("should create overlay extension config with Helm", func() {
				Expect(reconciler.Reconcile(ctx)).To(BeNil())

				var config overlayextensionconfig_v1alpha1.OverlayExtensionConfig
				Expect(k8sClient.Get(ctx, ctrl_client.ObjectKey{Name: cni.OverlayExtensionConfigName, Namespace: namespace}, &config)).To(BeNil())
				Expect(config.Labels).To(HaveLen(1))
				Expect(config.Labels[cni.ResourceManagedByLabel]).To(Equal(cni.ResourceManagedByHelmValue))
				Expect(config.Spec.ExtensionIPRange).To(Equal(subnetCIDR))
			})

			It("should create overlay extension config with addon", func() {
				// Create a new reconciler with addonMode set to true
				reconciler = cni.NewReconciler(azClient, k8sClient, recorder, &azure.CloudProviderConfig{
					SubscriptionID: "test-sub",
				}, appGw, agicPod, namespace, true)

				Expect(reconciler.Reconcile(ctx)).To(BeNil())

				var config overlayextensionconfig_v1alpha1.OverlayExtensionConfig
				Expect(k8sClient.Get(ctx, ctrl_client.ObjectKey{Name: cni.OverlayExtensionConfigName, Namespace: namespace}, &config)).To(BeNil())
				Expect(config.Labels).To(HaveLen(1))
				Expect(config.Labels[cni.ResourceManagedByLabel]).To(Equal(cni.ResourceManagedByAddonValue))
				Expect(config.Spec.ExtensionIPRange).To(Equal(subnetCIDR))
			})
		})

		It("should return error if failed to get subnet", func() {
			azClient.GetSubnetFunc = func(subnetID string) (n.Subnet, error) {
				return n.Subnet{}, errors.New("failed to get subnet")
			}

			err := reconciler.Reconcile(ctx)
			Expect(err).To(Not(BeNil()))
			Expect(err.Error()).To(ContainSubstring("failed to get subnet"))
		})
	})

	When("a non-overlay cluster is upgraded to Overlay CNI", func() {
		BeforeEach(func() {
			azClient.GetSubnetFunc = func(subnetID string) (n.Subnet, error) {
				return n.Subnet{SubnetPropertiesFormat: &n.SubnetPropertiesFormat{AddressPrefix: to.StringPtr(subnetCIDR)}}, nil
			}

			go func() {
				for {
					var config overlayextensionconfig_v1alpha1.OverlayExtensionConfig
					if err := k8sClient.Get(ctx, ctrl_client.ObjectKey{Name: cni.OverlayExtensionConfigName, Namespace: namespace}, &config); err != nil {
						time.Sleep(1 * time.Second)
						continue
					}
					config.Status.State = overlayextensionconfig_v1alpha1.Succeeded
					_ = k8sClient.Update(ctx, &config)
					break
				}
			}()
		})

		It("should create overlay extension config eventually", func() {
			Expect(reconciler.Reconcile(ctx)).To(BeNil())

			// It should not create OverlayExtensionConfig as the cluster is not overlay CNI
			var config overlayextensionconfig_v1alpha1.OverlayExtensionConfig
			Expect(k8sClient.Get(ctx, ctrl_client.ObjectKey{Name: cni.OverlayExtensionConfigName, Namespace: namespace}, &config)).To(Not(BeNil()))

			// Create NodeNetworkConfig so that cluster is considered as overlay CNI
			Expect(k8sClient.Create(ctx, &nodenetworkconfig_v1alpha.NodeNetworkConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node-network-config", Namespace: namespace},
				Spec:       nodenetworkconfig_v1alpha.NodeNetworkConfigSpec{},
			})).To(BeNil())

			Expect(reconciler.Reconcile(ctx)).To(BeNil())

			// It should create OverlayExtensionConfig as the cluster is overlay CNI
			Expect(k8sClient.Get(ctx, ctrl_client.ObjectKey{Name: cni.OverlayExtensionConfigName, Namespace: namespace}, &config)).To(BeNil())
		})
	})

	AfterEach(func() {
		cancel()
	})
})
