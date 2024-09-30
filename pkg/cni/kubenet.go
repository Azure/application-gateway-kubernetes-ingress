package cni

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
)

func (r *Reconciler) reconcileKubenetCniIfNeeded(cpConfig *azure.CloudProviderConfig, subnetID string) error {
	if cpConfig == nil || cpConfig.RouteTableName == "" {
		return nil
	}

	routeTableID := azure.RouteTableID(azure.SubscriptionID(cpConfig.SubscriptionID), azure.ResourceGroup(cpConfig.RouteTableResourceGroup), azure.ResourceName(cpConfig.RouteTableName))
	return r.armClient.ApplyRouteTable(subnetID, routeTableID)
}
