// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	//n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	//"k8s.io/api/extensions/v1beta1"

	//"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
)

// processIngressRules creates the sets of front end listeners and ports, and a map of azure config per listener for the given ingress.
func (c *appGwConfigBuilder) processIstioIngressRules(virtualService *v1alpha3.VirtualService, env environment.EnvVariables) (map[int32]interface{}, map[listenerIdentifier]listenerAzConfig) {
	frontendPorts := make(map[int32]interface{})
	listeners := make(map[listenerIdentifier]listenerAzConfig)
	for _, rule := range virtualService.Spec.HTTP {
		ruleFrontendPorts, ruleListeners := c.processIstioIngressRule(&rule, virtualService, env)
		for k, v := range ruleFrontendPorts {
			frontendPorts[k] = v
		}
		for k, v := range ruleListeners {
			listeners[k] = v
		}
	}
	return frontendPorts, listeners
}

func (c *appGwConfigBuilder) processIstioIngressRule(rule *v1alpha3.HTTPRoute, virtualService *v1alpha3.VirtualService, env environment.EnvVariables) (map[int32]interface{}, map[listenerIdentifier]listenerAzConfig) {
	frontendPorts := make(map[int32]interface{})
	listeners := make(map[listenerIdentifier]listenerAzConfig)
	/* TODO (rhea): expand this function without certificates */

	return frontendPorts, listeners
}
