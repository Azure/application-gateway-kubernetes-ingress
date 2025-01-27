package cni

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/pkg/errors"
)

func (r *Reconciler) reconcileKubenetCniIfNeeded(cpConfig *azure.CloudProviderConfig, subnetID string) error {
	if cpConfig == nil || cpConfig.RouteTableName == "" {
		return nil
	}

	if r.routeTableAttached {
		return nil
	}

	routeTableID := azure.RouteTableID(azure.SubscriptionID(cpConfig.SubscriptionID), azure.ResourceGroup(cpConfig.RouteTableResourceGroup), azure.ResourceName(cpConfig.RouteTableName))
	if err := r.armClient.ApplyRouteTable(subnetID, routeTableID); err != nil {
		return errors.Wrapf(err, "Unable to associate Application Gateway subnet '%s' with route table '%s' due to error (this is relevant for AKS clusters using 'Kubenet' network plugin)",
			subnetID,
			routeTableID)
	}

	r.routeTableAttached = true
	return nil
}
