// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"k8s.io/klog/v2"
)

func (c *appGwConfigBuilder) getIstioListenersPorts(cbCtx *ConfigBuilderContext) ([]n.ApplicationGatewayHTTPListener, map[Port]n.ApplicationGatewayFrontendPort, map[string]string) {
	publIPPorts := make(map[string]string)
	portsByNumber := cbCtx.ExistingPortsByNumber
	var listeners []n.ApplicationGatewayHTTPListener

	if cbCtx.EnvVariables.EnableIstioIntegration {
		for listenerID, config := range c.getListenerConfigsFromIstio(cbCtx.IstioGateways, cbCtx.IstioVirtualServices) {
			listener, port, err := c.newListener(cbCtx, listenerID, config.Protocol, portsByNumber)
			if err != nil {
				klog.Errorf("Failed creating listener %+v: %s", listenerID, err)
				continue
			}
			if listenerName, exists := publIPPorts[*port.Name]; exists && listenerID.UsePrivateIP {
				klog.Errorf("Can't assign port %s to Private IP Listener %s; already assigned to Public IP Listener %s", *port.Name, *listener.Name, listenerName)
				continue
			}

			if !listenerID.UsePrivateIP {
				publIPPorts[*port.Name] = *listener.Name
			}

			listeners = append(listeners, *listener)
			if _, exists := portsByNumber[Port(*port.Port)]; !exists {
				portsByNumber[Port(*port.Port)] = *port
			}
		}
	}
	return listeners, portsByNumber, publIPPorts
}
