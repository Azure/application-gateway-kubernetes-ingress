package controller

import (
	"github.com/knative/pkg/apis/istio/v1alpha3"
	v1 "k8s.io/api/core/v1"
)

// GetEndpointsForVirtualService returns all endpoints for the given virtual service.
func GetEndpointsForVirtualService(virtualService v1alpha3.VirtualService) v1.EndpointSubset {
	return v1.EndpointSubset{}
}
