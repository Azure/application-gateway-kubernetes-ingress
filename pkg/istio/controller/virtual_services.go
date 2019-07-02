package controller

import "github.com/knative/pkg/apis/istio/v1alpha3"

// GetVirtualServicesForGateway returns the VirtualServices for the provided gateway
func GetVirtualServicesForGateway(gateway v1alpha3.Gateway) []*v1alpha3.VirtualService {
	virtualServices := make([]*v1alpha3.VirtualService, 0)
	allVirtualServices := make([]*v1alpha3.VirtualService, 0) /* TO DO - get all virtual services and replace this */
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
