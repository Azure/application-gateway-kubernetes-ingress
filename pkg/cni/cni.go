package cni

import (
	"context"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler reconciles the resources required to configure
// CNI on the AKS cluster.
type Reconciler struct {
	armClient azure.AzClient
	client    client.Client
	cpConfig  *azure.CloudProviderConfig
	appGw     n.ApplicationGateway
	namespace string
	addonMode bool

	reconciledKubenetCNI bool
	reconciledOverlayCNI bool
}

func NewReconciler(armClient azure.AzClient,
	client client.Client,
	recorder record.EventRecorder,
	cpConfig *azure.CloudProviderConfig,
	appGw n.ApplicationGateway,
	agicPod *v1.Pod,
	namespace string,
	addonMode bool) *Reconciler {
	return &Reconciler{
		armClient:            armClient,
		client:               client,
		cpConfig:             cpConfig,
		appGw:                appGw,
		namespace:            namespace,
		addonMode:            addonMode,
		reconciledKubenetCNI: false,
		reconciledOverlayCNI: false,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	subnetID := *(*r.appGw.GatewayIPConfigurations)[0].Subnet.ID

	if err := r.reconcileKubenetCniIfNeeded(r.cpConfig, subnetID); err != nil {
		return errors.Wrap(err, "failed to reconcile kubenet CNI")
	}

	if err := r.reconcileOverlayCniIfNeeded(ctx, subnetID); err != nil {
		return errors.Wrap(err, "failed to reconcile overlay CNI")
	}

	return nil
}
