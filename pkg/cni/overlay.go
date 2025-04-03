package cni

import (
	"context"
	"fmt"
	"time"

	nodenetworkconfig_v1alpha "github.com/Azure/azure-container-networking/crd/nodenetworkconfig/api/v1alpha"
	overlayextensionconfig_v1alpha1 "github.com/Azure/azure-container-networking/crd/overlayextensionconfig/api/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ResourceManagedByLabel      = "app.kubernetes.io/managed-by"
	ResourceManagedByAddonValue = "ingress-appgw-addon"
	ResourceManagedByHelmValue  = "ingress-appgw-helm"
)

const (
	// PodNetworkTypeLabel is the name of the label on NNCs to tell what mode the network is in.
	PodNetworkTypeLabel = "kubernetes.azure.com/podnetwork-type"

	// OverlayExtensionConfigName is the name of the overlay extension config resource
	OverlayExtensionConfigName = "agic-overlay-extension-config"

	// OverlayConfigReconcileTimeout for checking overlay extension config status
	OverlayConfigReconcileTimeout = 30 * time.Second

	// OverlayConfigReconcilePollInterval for checking overlay extension config status
	OverlayConfigReconcilePollInterval = 2 * time.Second
)

func (r *Reconciler) reconcileOverlayCniIfNeeded(ctx context.Context, subnetID string) error {
	if r.reconciledOverlayCNI {
		return nil
	}

	isOverlay, err := r.isClusterOverlayCNI(ctx)
	if err != nil {
		return errors.New("failed to check if cluster is using overlay CNI")
	}

	if !isOverlay {
		return nil
	}

	klog.Infof("Cluster is using overlay CNI, using subnetID %q for application gateway", subnetID)
	subnet, err := r.armClient.GetSubnet(subnetID)
	if err != nil {
		return errors.Wrap(err, "failed to get subnet")
	}

	var subnetCIDR string
	if subnet.AddressPrefix != nil {
		subnetCIDR = *subnet.AddressPrefix
	} else if subnet.AddressPrefixes != nil && len(*subnet.AddressPrefixes) > 0 {
		subnetCIDR = (*subnet.AddressPrefixes)[0]
	} else {
		return errors.New("subnet does not have an address prefix(es)")
	}

	klog.Infof("Using subnet prefix %q", subnetCIDR)
	err = r.reconcileOverlayExtensionConfig(ctx, subnetCIDR)
	if err != nil {
		return errors.Wrap(err, "failed to reconcile overlay resources")
	}

	r.reconciledOverlayCNI = true
	return nil
}

func (r *Reconciler) isClusterOverlayCNI(ctx context.Context) (bool, error) {
	var nodeNetworkConfigs nodenetworkconfig_v1alpha.NodeNetworkConfigList
	if err := r.client.List(ctx, &nodeNetworkConfigs); err != nil {
		if meta.IsNoMatchError(err) {
			return false, nil
		}

		return false, errors.Wrap(err, "failed to list node network configs")
	}

	// if any NNCs are overlay then this cluster is using CNI Overlay
	for _, nnc := range nodeNetworkConfigs.Items {
		if val, ok := nnc.Labels[PodNetworkTypeLabel]; ok && val == "overlay" {
			return true, nil
		}
	}
	return false, nil
}

func (r *Reconciler) reconcileOverlayExtensionConfig(ctx context.Context, subnetCIDR string) error {
	var config overlayextensionconfig_v1alpha1.OverlayExtensionConfig
	if err := r.client.Get(ctx, client.ObjectKey{
		Name:      OverlayExtensionConfigName,
		Namespace: r.namespace,
	}, &config); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get overlay extension config")
		}

		return r.createOverlayExtensionConfig(ctx, subnetCIDR)
	}

	if config.Spec.ExtensionIPRange == subnetCIDR {
		return r.checkOverlayExtensionConfigStatus(ctx)
	}

	klog.Infof("Updating overlay extension config with subnet CIDR %s", subnetCIDR)

	// Delete the existing config and create a new one
	if err := r.client.Delete(ctx, &config); err != nil {
		return errors.Wrap(err, "failed to delete overlay extension config")
	}

	return r.createOverlayExtensionConfig(ctx, subnetCIDR)
}

func (r *Reconciler) createOverlayExtensionConfig(ctx context.Context, subnetCIDR string) error {
	managedByValue := ResourceManagedByHelmValue
	if r.addonMode {
		managedByValue = ResourceManagedByAddonValue
	}

	config := overlayextensionconfig_v1alpha1.OverlayExtensionConfig{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      OverlayExtensionConfigName,
			Namespace: r.namespace,
			Labels: map[string]string{
				ResourceManagedByLabel: managedByValue,
			},
		},
		Spec: overlayextensionconfig_v1alpha1.OverlayExtensionConfigSpec{
			ExtensionIPRange: subnetCIDR,
		},
	}

	klog.Infof("Creating overlay extension config with subnet CIDR %s", subnetCIDR)
	if err := r.client.Create(ctx, &config); err != nil {
		return errors.Wrap(err, "failed to create overlay extension config")
	}

	return r.checkOverlayExtensionConfigStatus(ctx)
}

func (r *Reconciler) checkOverlayExtensionConfigStatus(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, OverlayConfigReconcileTimeout)
	defer cancel()

	checkStatus := func() (bool, error) {
		var config overlayextensionconfig_v1alpha1.OverlayExtensionConfig
		if err := r.client.Get(ctx, client.ObjectKey{
			Name:      OverlayExtensionConfigName,
			Namespace: r.namespace,
		}, &config); err != nil {
			return false, errors.Wrap(err, "failed to get overlay extension config")
		}

		switch config.Status.State {
		case overlayextensionconfig_v1alpha1.Succeeded:
			klog.Infof("Overlay extension config is ready")
			return true, nil
		case overlayextensionconfig_v1alpha1.Failed:
			return true, fmt.Errorf("overlay extension config failed with error: %s", config.Status.Message)
		}

		klog.Infof("Waiting for overlay extension config to be ready")
		return false, nil
	}

	// Initial check
	done, err := checkStatus()
	if done {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return errors.New("timed out waiting for overlay extension config to be ready")
		case <-time.After(OverlayConfigReconcilePollInterval):
			done, err := checkStatus()
			if done {
				return err
			}
		}
	}
}
