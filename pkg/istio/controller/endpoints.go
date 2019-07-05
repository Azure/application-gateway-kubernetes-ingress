package controller

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	v1 "k8s.io/api/core/v1"
)

// GetEndpointsForVirtualService returns all endpoints for the given virtual service.
func GetEndpointsForVirtualService(virtualService v1alpha3.VirtualService) v1.EndpointSubset {
	var endpointSubset v1.EndpointSubset
	addresses := make([]v1.EndpointAddress, len(virtualService.Spec.Hosts))
	for _, host := range virtualService.Spec.Hosts {
		var newAddress v1.EndpointAddress
		newAddress.IP = host
		addresses = append(addresses, newAddress)
	}
	endpointSubset.Addresses = addresses
	var endpointsLogging []string
	for _, endpoint := range addresses {
		endpointsLogging = append(endpointsLogging, fmt.Sprintf(endpoint.IP))
	}
	glog.V(5).Infof("Found Endpoints: %+v", strings.Join(endpointsLogging, ","))
	print(strings.Join(endpointsLogging, ","))
	return endpointSubset
}
