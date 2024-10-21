package cni

import (
	"context"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler reconciles the resources required to configure
// CNI on the AKS cluster.
type Reconciler struct {
	armClient azure.AzClient
	client    client.Client
	namespace string
}

func ReconcileCNI(ctx context.Context, armClient azure.AzClient, client client.Client, namespace string, cpConfig *azure.CloudProviderConfig, appGw n.ApplicationGateway) error {
	r := &Reconciler{
		armClient: armClient,
		client:    client,
		namespace: namespace,
	}

	return r.Reconcile(ctx, cpConfig, appGw)
}

func (r *Reconciler) Reconcile(ctx context.Context, cpConfig *azure.CloudProviderConfig, appGw n.ApplicationGateway) error {
	subnetID := *(*appGw.GatewayIPConfigurations)[0].Subnet.ID

	if err := r.reconcileOverlayCniIfNeeded(ctx, subnetID); err != nil {
		return errors.Wrap(err, "failed to reconcile overlay CNI")
	}

	if err := r.reconcileKubenetCniIfNeeded(cpConfig, subnetID); err != nil {
		return errors.Wrap(err, "failed to reconcile kubenet CNI")
	}
	return nil
}
