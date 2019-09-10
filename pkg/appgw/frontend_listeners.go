// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

// getListeners constructs the unique set of App Gateway HTTP listeners across all ingresses.
func (c *appGwConfigBuilder) getListeners(cbCtx *ConfigBuilderContext) (*[]n.ApplicationGatewayHTTPListener, *[]n.ApplicationGatewayFrontendPort) {
	if c.mem.listeners != nil && c.mem.ports != nil {
		return c.mem.listeners, c.mem.ports
	}

	publIPPorts := make(map[string]string)
	portSet := make(map[string]interface{})
	var listeners []n.ApplicationGatewayHTTPListener
	var ports []n.ApplicationGatewayFrontendPort

	if cbCtx.EnvVariables.EnableIstioIntegration {
		for listenerID, config := range c.getListenerConfigsFromIstio(cbCtx.IstioGateways, cbCtx.IstioVirtualServices) {
			listener, port, err := c.newListener(cbCtx, listenerID, config.Protocol)
			if err != nil {
				glog.Errorf("Failed creating listener %+v: %s", listenerID, err)
				continue
			}
			if listenerName, exists := publIPPorts[*port.Name]; exists && listenerID.UsePrivateIP {
				glog.Errorf("Can't assign port %s to Private IP Listener %s; already assigned to Public IP Listener %s", *port.Name, *listener.Name, listenerName)
				continue
			}

			if !listenerID.UsePrivateIP {
				publIPPorts[*port.Name] = *listener.Name
			}

			listeners = append(listeners, *listener)
			if _, exists := portSet[*port.Name]; !exists {
				portSet[*port.Name] = nil
				ports = append(ports, *port)
			}
		}
	}

	for listenerID, config := range c.getListenerConfigs(cbCtx) {
		listener, port, err := c.newListener(cbCtx, listenerID, config.Protocol)
		if err != nil {
			glog.Errorf("Failed creating listener %+v: %s", listenerID, err)
			continue
		}

		if listenerName, exists := publIPPorts[*port.Name]; exists && listenerID.UsePrivateIP {
			glog.Errorf("Can't assign port %s to Private IP Listener %s; already assigned to Public IP Listener %s", *port.Name, *listener.Name, listenerName)
			continue
		}

		if !listenerID.UsePrivateIP {
			publIPPorts[*port.Name] = *listener.Name
		}

		if config.Protocol == n.HTTPS {
			sslCertificateID := c.appGwIdentifier.sslCertificateID(config.Secret.secretFullName())
			listener.SslCertificate = resourceRef(sslCertificateID)
		}
		listeners = append(listeners, *listener)
		if _, exists := portSet[*port.Name]; !exists {
			portSet[*port.Name] = nil
			ports = append(ports, *port)
		}
	}

	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		er := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)

		// Listeners we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
		existingBlacklisted, existingNonBlacklisted := er.GetBlacklistedListeners()

		brownfield.LogListeners(existingBlacklisted, existingNonBlacklisted, listeners)

		// MergeListeners would produce unique list of listeners based on Name. Blacklisted listeners,
		// which have the same name as a managed listeners would be overwritten.
		listeners = brownfield.MergeListeners(existingBlacklisted, listeners)
	}

	sort.Sort(sorter.ByListenerName(listeners))
	sort.Sort(sorter.ByFrontendPortName(ports))

	// Since getListeners() would be called multiple times within the life cycle of a Process(Event)
	// we cache the results of this function in what would be final place to store the Listeners.
	c.mem.listeners = &listeners
	c.mem.ports = &ports
	return &listeners, &ports
}

// getListenerConfigs creates an intermediary representation of the listener configs based on the passed list of ingresses
func (c *appGwConfigBuilder) getListenerConfigs(cbCtx *ConfigBuilderContext) map[listenerIdentifier]listenerAzConfig {
	if c.mem.listenerConfigs != nil {
		return *c.mem.listenerConfigs
	}

	// TODO(draychev): Emit an error event if 2 namespaces define different TLS for the same domain!
	allListeners := make(map[listenerIdentifier]listenerAzConfig)
	for _, ingress := range cbCtx.IngressList {
		glog.V(5).Infof("Processing Rules for Ingress: %s/%s", ingress.Namespace, ingress.Name)
		azListenerConfigs := c.getListenersFromIngress(ingress, cbCtx.EnvVariables)
		for listenerID, azConfig := range azListenerConfigs {
			allListeners[listenerID] = azConfig
		}
	}

	// App Gateway must have at least one listener - the default one!
	if len(allListeners) == 0 {
		allListeners[defaultFrontendListenerIdentifier()] = listenerAzConfig{
			// Default protocol
			Protocol: n.HTTP,
		}
	}

	c.mem.listenerConfigs = &allListeners
	return allListeners
}

func (c *appGwConfigBuilder) newListener(cbCtx *ConfigBuilderContext, listenerID listenerIdentifier, protocol n.ApplicationGatewayProtocol) (*n.ApplicationGatewayHTTPListener, *n.ApplicationGatewayFrontendPort, error) {
	frontIPConfiguration := *LookupIPConfigurationByType(c.appGw.FrontendIPConfigurations, listenerID.UsePrivateIP)

	portName := generateFrontendPortName(listenerID.FrontendPort)
	frontendPort := n.ApplicationGatewayFrontendPort{
		Etag: to.StringPtr("*"),
		Name: &portName,
		ID:   to.StringPtr(c.appGwIdentifier.frontendPortID(portName)),
		ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
			Port: to.Int32Ptr(int32(listenerID.FrontendPort)),
		},
	}

	listenerName := generateListenerName(listenerID)
	listener := n.ApplicationGatewayHTTPListener{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(listenerName),
		ID:   to.StringPtr(c.appGwIdentifier.listenerID(listenerName)),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			// TODO: expose this to external configuration
			FrontendIPConfiguration: resourceRef(*frontIPConfiguration.ID),
			FrontendPort:            resourceRef(*frontendPort.ID),
			Protocol:                protocol,
			HostName:                &listenerID.HostName,
		},
	}
	return &listener, &frontendPort, nil
}

func (c *appGwConfigBuilder) groupListenersByListenerIdentifier(cbCtx *ConfigBuilderContext) map[listenerIdentifier]*n.ApplicationGatewayHTTPListener {
	listeners, ports := c.getListeners(cbCtx)
	portsById := make(map[string]n.ApplicationGatewayFrontendPort)
	for _, port := range *ports {
		portsById[*port.ID] = port
	}

	listenersByID := make(map[listenerIdentifier]*n.ApplicationGatewayHTTPListener)
	// Update the listenerMap with the final listener lists
	for idx, listener := range *listeners {
		port := portsById[*listener.FrontendPort.ID]
		listenerID := listenerIdentifier{
			HostName:     *listener.HostName,
			FrontendPort: Port(*port.Port),
			UsePrivateIP: IsPrivateIPConfiguration(LookupIPConfigurationByID(c.appGw.FrontendIPConfigurations, listener.FrontendIPConfiguration.ID)),
		}
		listenersByID[listenerID] = &((*listeners)[idx])
	}

	return listenersByID
}
