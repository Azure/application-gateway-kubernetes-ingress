package cni_test

import (
	"context"
	"errors"

	nodenetworkconfig_v1alpha "github.com/Azure/azure-container-networking/crd/nodenetworkconfig/api/v1alpha"
	overlayextensionconfig_v1alpha1 "github.com/Azure/azure-container-networking/crd/overlayextensionconfig/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl_client "sigs.k8s.io/controller-runtime/pkg/client"
	ctrl_client_fake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/cni"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8s"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

var _ = Describe("Overlay CNI", func() {
	var ctx = context.TODO()
	var azClient *azure.FakeAzClient
	var k8sClient ctrl_client.Client
	var namespace = "test-namespace"
	var subnetCIDR = "10.0.0.0/16"
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
		azClient.ApplyRouteTableFunc = func(subnetID string, routeTableID string) error {
			Fail("ApplyRouteTable should not be called")
			return nil
		}

		scheme, _ := k8s.NewScheme()
		k8sClient = ctrl_client_fake.NewClientBuilder().WithScheme(scheme).Build()

		config := &nodenetworkconfig_v1alpha.NodeNetworkConfig{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "test-node-network-config",
				Namespace: namespace,
			},
			Spec: nodenetworkconfig_v1alpha.NodeNetworkConfigSpec{},
		}
		err := k8sClient.Create(ctx, config)
		Expect(err).To(BeNil())
	})

	Context("reconcileOverlayCniIfNeeded", func() {
		It("should create overlay extension config with addon if controller is addon", func() {
			azClient.GetSubnetFunc = func(subnetID string) (n.Subnet, error) {
				return n.Subnet{
					SubnetPropertiesFormat: &n.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr(subnetCIDR),
					},
				}, nil
			}

			err := cni.ReconcileCNI(context.TODO(), azClient, k8sClient, namespace, nil, appGw, true)
			Expect(err).To(BeNil())

			var config overlayextensionconfig_v1alpha1.OverlayExtensionConfig
			err = k8sClient.Get(ctx, ctrl_client.ObjectKey{
				Name:      cni.OverlayExtensionConfigName,
				Namespace: namespace,
			}, &config)
			Expect(err).To(BeNil())

			Expect(config.Labels).To(HaveLen(1))
			Expect(config.Labels[cni.ResourceManagedByLabel]).To(Equal(cni.ResourceManagedByAddonValue))
			Expect(config.Spec.ExtensionIPRange).To(Equal(subnetCIDR))
		})

		It("should create overlay extension config with addon if controller is addon", func() {
			azClient.GetSubnetFunc = func(subnetID string) (n.Subnet, error) {
				return n.Subnet{
					SubnetPropertiesFormat: &n.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr(subnetCIDR),
					},
				}, nil
			}

			err := cni.ReconcileCNI(context.TODO(), azClient, k8sClient, namespace, nil, appGw, false)
			Expect(err).To(BeNil())

			var config overlayextensionconfig_v1alpha1.OverlayExtensionConfig
			err = k8sClient.Get(ctx, ctrl_client.ObjectKey{
				Name:      cni.OverlayExtensionConfigName,
				Namespace: namespace,
			}, &config)
			Expect(err).To(BeNil())

			Expect(config.Labels).To(HaveLen(1))
			Expect(config.Labels[cni.ResourceManagedByLabel]).To(Equal(cni.ResourceManagedByHelmValue))
			Expect(config.Spec.ExtensionIPRange).To(Equal(subnetCIDR))
		})

		It("should return error if failed to get subnet", func() {
			azClient.GetSubnetFunc = func(subnetID string) (n.Subnet, error) {
				return n.Subnet{}, errors.New("failed to get subnet")
			}

			err := cni.ReconcileCNI(context.TODO(), azClient, k8sClient, namespace, nil, appGw, false)
			Expect(err).ToNot(BeNil())
		})
	})
})
