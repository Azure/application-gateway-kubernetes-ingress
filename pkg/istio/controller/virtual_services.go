package controller

import "github.com/knative/pkg/apis/istio/v1alpha3"

// GetVirtualServicesForGateway returns the VirtualServices for the provided gateway
func GetVirtualServicesForGateway(gateway v1alpha3.Gateway) []*v1alpha3.VirtualService {
	return []*v1alpha3.VirtualService{}
}
