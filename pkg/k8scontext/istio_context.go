// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import "github.com/knative/pkg/apis/istio/v1alpha3"

// ListIstioGateways returns a list of discovered Istio Gateways
func (c *Context) ListIstioGateways() []*v1alpha3.Gateway {
	var gateways []*v1alpha3.Gateway
	for _, gateway := range c.Caches.IstioGateway.List() {
		gateways = append(gateways, gateway.(*v1alpha3.Gateway))
	}
	return gateways
}

// ListIstioVirtualServices returns a list of discovered Istio Virtual Services
func (c *Context) ListIstioVirtualServices() []*v1alpha3.VirtualService {
	var virtualServices []*v1alpha3.VirtualService
	for _, virtualService := range c.Caches.IstioVirtualService.List() {
		virtualServices = append(virtualServices, virtualService.(*v1alpha3.VirtualService))
	}
	return virtualServices
}
