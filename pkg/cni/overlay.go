package cni

import (
	"context"

	nodenetworkconfig_v1alpha "github.com/Azure/azure-container-networking/crd/nodenetworkconfig/api/v1alpha"
	overlayextensionconfig_v1alpha1 "github.com/Azure/azure-container-networking/crd/overlayextensionconfig/api/v1alpha1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// OverlayExtensionConfigName is the name of the overlay extension config resource
	OverlayExtensionConfigName = "agic-overlay-extension-config"
)

func (r *Reconciler) reconcileOverlayCniIfNeeded(ctx context.Context, subnetID string) error {
	isOverlay, err := r.isClusterOverlayCNI(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to check if cluster is using overlay CNI")
	}

	if !isOverlay {
		return nil
	}

	klog.Infof("Cluster is using overlay CNI, using subnetID %q for application gateway", subnetID)
	subnet, err := r.armClient.GetSubnet(subnetID)
	if err != nil {
		return errors.Wrap(err, "failed to get subnet")
	}

	subnetCIDR := *subnet.AddressPrefix

	klog.Infof("Cluster is using overlay CNI, reconciling overlay resources")
	err = r.reconcileOverlayExtensionConfig(ctx, subnetCIDR)
	if err != nil {
		return errors.Wrap(err, "failed to reconcile overlay resources")
	}

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

	return len(nodeNetworkConfigs.Items) > 0, nil
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

		config = overlayextensionconfig_v1alpha1.OverlayExtensionConfig{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      OverlayExtensionConfigName,
				Namespace: r.namespace,
			},
			Spec: overlayextensionconfig_v1alpha1.OverlayExtensionConfigSpec{
				ExtensionIPRange: subnetCIDR,
			},
		}

		klog.Infof("Creating overlay extension config with subnet CIDR %s", subnetCIDR)
		err := r.client.Create(ctx, &config)
		if err != nil {
			return errors.Wrap(err, "failed to create overlay extension config")
		}
		return nil
	}

	config.Spec.ExtensionIPRange = subnetCIDR
	klog.Infof("Updating overlay extension config with subnet CIDR %s", subnetCIDR)
	err := r.client.Update(ctx, &config)
	if err != nil {
		return errors.Wrap(err, "failed to update overlay extension config")
	}
	return nil
}
