package cni

import (
	"context"

	overlayv1alpha1 "github.com/Azure/azure-container-networking/crd/overlayextensionconfig/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CleanupOverlayExtensionConfigs(k8sClient client.Client, namespace string, addonMode bool) error {
	// Define a label selector for filtering the resources to delete.
	managedByValue := ResourceManagedByHelmValue
	if addonMode {
		managedByValue = ResourceManagedByAddonValue
	}

	// Perform cleanup of OverlayExtensionConfig resources.
	if err := cleanupOverlayExtensionConfigs(k8sClient, namespace, managedByValue); err != nil {
		klog.Errorf("Error cleaning up OverlayExtensionConfig resources: %v", err)
		return err
	}

	klog.Infof("Cleanup completed successfully.")
	return nil
}

// cleanupOverlayExtensionConfigs lists and deletes OverlayExtensionConfig resources in the given namespace
// that match the provided label selector. If the CRD is not present, it logs a warning and returns nil.
func cleanupOverlayExtensionConfigs(c client.Client, namespace string, label string) error {
	ctx := context.Background()

	// Create an empty list to hold OverlayExtensionConfig resources.
	var overlayList overlayv1alpha1.OverlayExtensionConfigList

	// List the resources with the provided namespace and label selector.
	if err := c.List(ctx, &overlayList,
		client.InNamespace(namespace),
		client.MatchingLabels(map[string]string{ResourceManagedByLabel: label})); err != nil {
		// If the API server does not recognize the CRD, skip cleanup.
		if meta.IsNoMatchError(err) {
			klog.Warning("CRD OverlayExtensionConfig not found in the cluster. Skipping cleanup.")
			return nil
		}
		return err
	}

	// If no resources are found, log and exit.
	if len(overlayList.Items) == 0 {
		klog.Infof("No OverlayExtensionConfig resources found in namespace %q with labels %q", namespace, label)
		return nil
	}

	// Iterate through and delete each OverlayExtensionConfig resource.
	var deletionError error
	for _, item := range overlayList.Items {
		klog.Infof("Deleting OverlayExtensionConfig: %q", item.Name)
		if err := c.Delete(ctx, &item, &client.DeleteOptions{}); err != nil {
			klog.Errorf("Error deleting resource %q: %v", item.Name, err)
			deletionError = err
		} else {
			klog.Infof("Successfully deleted resource %q", item.Name)
		}
	}

	return deletionError
}
