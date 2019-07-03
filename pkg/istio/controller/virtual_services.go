package controller

import "github.com/knative/pkg/apis/istio/v1alpha3"
import "github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"

// GetVirtualServicesForGateway returns the VirtualServices for the provided gateway
func GetVirtualServicesForGateway(gateway v1alpha3.Gateway, c *k8scontext.Context) []*v1alpha3.VirtualService {
	virtualServices := make([]*v1alpha3.VirtualService, 0)
	allVirtualServices := c.ListIstioVirtualServices()
	gatewayName := gateway.Name
	for _, service := range allVirtualServices {
		for _, serviceGateway := range service.Spec.Gateways {
			if gatewayName == serviceGateway {
				virtualServices = append(virtualServices, service)
			}
		}
	}
	return virtualServices
}
