package controller

import "github.com/knative/pkg/apis/istio/v1alpha3"

// GetGateways returns all Istio Gateways that are annotated.
func GetGateways(annotation string) []*v1alpha3.Gateway {
	return []*v1alpha3.Gateway{}
}
