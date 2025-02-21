package cni_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/cni"
	overlayv1alpha1 "github.com/Azure/azure-container-networking/crd/overlayextensionconfig/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CleanupOverlayExtensionConfigs", func() {
	var (
		scheme    *runtime.Scheme
		namespace string
		label     string
	)

	BeforeEach(func() {
		namespace = "default"
		// Use the Helm-managed value as default.
		label = cni.ResourceManagedByHelmValue

		scheme = runtime.NewScheme()
		err := overlayv1alpha1.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when the CRD is not present", func() {
		It("should log a warning and succeed", func() {
			// Use a fake client that simulates a NoMatch error during List.
			fakeClient := &FakeClient{
				ListFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
					// Simulate that the API server does not recognize the CRD.
					return &meta.NoResourceMatchError{
						PartialResource: schema.GroupVersionResource{Group: "acn.azure.com", Resource: "overlayextensionconfigs"},
					}
				},
			}

			err := cni.CleanupOverlayExtensionConfigs(fakeClient, namespace, false)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when no OverlayExtensionConfig resources are found", func() {
		It("should succeed without deleting anything", func() {
			// Build a fake client with no objects.
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			err := cni.CleanupOverlayExtensionConfigs(k8sClient, namespace, false)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when OverlayExtensionConfig resources are present", func() {
		const resourceName = "test-resource"
		var k8sClient client.Client
		BeforeEach(func() {
			// Create a resource with the expected label.
			resource := &overlayv1alpha1.OverlayExtensionConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
					Labels: map[string]string{
						cni.ResourceManagedByLabel: label,
					},
				},
			}
			// Build the fake client with this object.
			k8sClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(resource).Build()
		})

		It("should delete the resource successfully", func() {
			err := cni.CleanupOverlayExtensionConfigs(k8sClient, namespace, false)
			Expect(err).NotTo(HaveOccurred())

			// Verify that the resource was deleted by attempting to get it.
			retrieved := &overlayv1alpha1.OverlayExtensionConfig{}
			err = k8sClient.Get(context.Background(), client.ObjectKey{Name: resourceName, Namespace: namespace}, retrieved)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when deletion fails", func() {
		It("should return an error", func() {
			// Use a fake client that returns an error when Delete is called.
			fakeClient := &FakeClient{
				ListFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
					// Populate the list with one resource.
					overlayList, ok := list.(*overlayv1alpha1.OverlayExtensionConfigList)
					if !ok {
						return fmt.Errorf("unexpected type")
					}
					overlayList.Items = []overlayv1alpha1.OverlayExtensionConfig{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "fail-resource",
								Namespace: namespace,
								Labels: map[string]string{
									cni.ResourceManagedByLabel: label,
								},
							},
						},
					}
					return nil
				},
				DeleteFunc: func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
					return errors.New("deletion error")
				},
			}

			err := cni.CleanupOverlayExtensionConfigs(fakeClient, namespace, false)
			Expect(err).To(MatchError("deletion error"))
		})
	})
})

// FakeClient implements client.Client for testing purposes.
// We only implement the List and Delete methods.
type FakeClient struct {
	client.Client

	ListFunc   func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
	DeleteFunc func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
}

func (f *FakeClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if f.ListFunc != nil {
		return f.ListFunc(ctx, list, opts...)
	}
	return nil
}

func (f *FakeClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if f.DeleteFunc != nil {
		return f.DeleteFunc(ctx, obj, opts...)
	}
	return nil
}
