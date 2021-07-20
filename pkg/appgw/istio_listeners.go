// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	"k8s.io/klog/v2"
)

func (c *appGwConfigBuilder) getListenerConfigsFromIstio(istioGateways []*v1alpha3.Gateway, istioVirtualServices []*v1alpha3.VirtualService) map[listenerIdentifier]listenerAzConfig {
	knownHosts := make(map[string]interface{})
	for _, virtualService := range istioVirtualServices {
		for _, host := range virtualService.Spec.Hosts {
			knownHosts[host] = nil
		}
	}

	allListeners := make(map[listenerIdentifier]listenerAzConfig)
	for _, igwy := range istioGateways {
		for _, server := range igwy.Spec.Servers {
			if server.Port.Protocol != v1alpha3.ProtocolHTTP {
				klog.Infof("[istio] AGIC does not support Gateway with Server.Port.Protocol=%+v", server.Port.Protocol)
				continue
			}
			for _, host := range server.Hosts {
				if _, exist := knownHosts[host]; !exist {
					continue
				}
				listenerID := listenerIdentifier{
					FrontendPort: Port(server.Port.Number),
					HostNames:    [5]string{host},
				}
				allListeners[listenerID] = listenerAzConfig{Protocol: n.HTTP}
			}
		}
	}

	// App Gateway must have at least one listener - the default one!
	if len(allListeners) == 0 {
		// TODO(aksgupta): refactor to get environment variable
		allListeners[defaultFrontendListenerIdentifier(false)] = listenerAzConfig{
			// Default protocol
			Protocol: n.HTTP,
		}
	}

	return allListeners
}
