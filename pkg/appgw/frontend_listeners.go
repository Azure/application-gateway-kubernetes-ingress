// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

// getListeners constructs the unique set of App Gateway HTTP listeners across all ingresses.
func (c *appGwConfigBuilder) getListeners(cbCtx *ConfigBuilderContext) (*[]n.ApplicationGatewayHTTPListener, *[]n.ApplicationGatewayFrontendPort) {
	if c.mem.listeners != nil && c.mem.ports != nil {
		return c.mem.listeners, c.mem.ports
	}

	publIPPorts := make(map[string]string)
	portsByNumber := cbCtx.ExistingPortsByNumber
	var listeners []n.ApplicationGatewayHTTPListener

	if portsByNumber == nil {
		portsByNumber = make(map[Port]n.ApplicationGatewayFrontendPort)
	}

	if cbCtx.EnvVariables.EnableIstioIntegration {
		listeners, portsByNumber, publIPPorts = c.getIstioListenersPorts(cbCtx)
	}

	for listenerID, config := range c.getListenerConfigs(cbCtx) {
		listener, port, err := c.newListener(cbCtx, listenerID, config.Protocol, portsByNumber)
		if err != nil {
			glog.Errorf("Failed creating listener %+v: %s", listenerID, err)
			continue
		}

		if listenerName, exists := publIPPorts[*port.Name]; exists && listenerID.UsePrivateIP {
			glog.Errorf("Can't assign port %s to Private IP Listener %s; already assigned to Public IP Listener %s; Will not create listener %+v", *port.Name, *listener.Name, listenerName, listenerID)
			continue
		}

		if !listenerID.UsePrivateIP {
			publIPPorts[*port.Name] = *listener.Name
		}

		// newlistener created a new port; Add it to the set
		if _, exists := portsByNumber[Port(*port.Port)]; !exists {
			portsByNumber[Port(*port.Port)] = *port
		}

		if config.Protocol == n.HTTPS {
			sslCertificateID := c.appGwIdentifier.sslCertificateID(config.Secret.secretFullName())
			listener.SslCertificate = resourceRef(sslCertificateID)
		}
		if config.FirewallPolicy != "" {
			listener.FirewallPolicy = &n.SubResource{ID: to.StringPtr(config.FirewallPolicy)}
		}
		listeners = append(listeners, *listener)
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

	portIDs := make(map[string]interface{})
	// Cleanup unused ports
	for _, listener := range listeners {
		if listener.FrontendPort != nil && listener.FrontendPort.ID != nil {
			portIDs[*listener.FrontendPort.ID] = nil
		}
	}

	var ports []n.ApplicationGatewayFrontendPort
	for _, port := range portsByNumber {
		if _, exists := portIDs[*port.ID]; exists {
			ports = append(ports, port)
		}
	}

	sort.Sort(sorter.ByListenerName(listeners))
	sort.Sort(sorter.ByFrontendPortName(ports))

	// Since getListeners() would be called multiple times within the life cycle of a MutateAppGateway(Event)
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
		policy, err := annotations.WAFPolicy(ingress)
		if len(policy) > 0 {
			glog.V(5).Infof("Found WAF policy: %s", policy)
		} else {
			glog.Error("WAF policy is empty, check your annotation.")
		}
		azListenerConfigs := c.getListenersFromIngress(ingress, cbCtx.EnvVariables)
		for listenerID, azConfig := range azListenerConfigs {
			if cbCtx.EnvVariables.AttachWAFPolicyToListener || (err == nil && policy != "") {
				azConfig.FirewallPolicy = policy
			}
			allListeners[listenerID] = azConfig
		}
	}

	// App Gateway must have at least one listener - the default one!
	if len(allListeners) == 0 {
		listenerConfig := listenerAzConfig{
			// Default protocol
			Protocol: n.HTTP,
		}
		// See if we have an ingress annotated with a Firewall Policy; Attach it to the listener
		for _, ingress := range cbCtx.IngressList {
			if policy, err := annotations.WAFPolicy(ingress); err == nil && policy != "" {
				listenerConfig.FirewallPolicy = policy
				break
			}
		}
		allListeners[defaultFrontendListenerIdentifier()] = listenerConfig
	}

	c.mem.listenerConfigs = &allListeners
	return allListeners
}

func (c *appGwConfigBuilder) newListener(cbCtx *ConfigBuilderContext, listenerID listenerIdentifier, protocol n.ApplicationGatewayProtocol, portsByNumber map[Port]n.ApplicationGatewayFrontendPort) (*n.ApplicationGatewayHTTPListener, *n.ApplicationGatewayFrontendPort, error) {
	frontIPConfiguration := *LookupIPConfigurationByType(c.appGw.FrontendIPConfigurations, listenerID.UsePrivateIP)
	portNumber := listenerID.FrontendPort
	var frontendPort n.ApplicationGatewayFrontendPort
	var exists bool
	if frontendPort, exists = portsByNumber[portNumber]; !exists {
		portName := generateFrontendPortName(listenerID.FrontendPort)
		frontendPort = n.ApplicationGatewayFrontendPort{
			Etag: to.StringPtr("*"),
			Name: &portName,
			ID:   to.StringPtr(c.appGwIdentifier.frontendPortID(portName)),
			ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
				Port: to.Int32Ptr(int32(portNumber)),
			},
		}
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
			HostName:                nil,
			Hostnames:               &[]string{},

			// setting to default
			RequireServerNameIndication: to.BoolPtr(false),
		},
	}

	// Use only the 'Hostnames' field as application gateway allows either 'HostName' or 'Hostnames'
	if hostnames := listenerID.getHostNames(); len(hostnames) != 0 {
		if len(hostnames) == 1 {
			listener.HostName = &hostnames[0]
		} else {
			listener.Hostnames = &hostnames
		}
	}

	// Note: This field is only supported on V1 gateway.
	// For V1 gateway, set RequireServerNameIndication only when listener is HTTPS and is provided with a hostname.
	if (c.appGw.Sku.Tier == n.ApplicationGatewayTierStandard || c.appGw.Sku.Tier == n.ApplicationGatewayTierWAF) &&
		len(listenerID.HostName) > 0 &&
		protocol == n.HTTPS {
		listener.RequireServerNameIndication = to.BoolPtr(true)
	}

	return &listener, &frontendPort, nil
}

func (c *appGwConfigBuilder) groupListenersByListenerIdentifier(cbCtx *ConfigBuilderContext) map[listenerIdentifier]*n.ApplicationGatewayHTTPListener {
	listeners, ports := c.getListeners(cbCtx)
	portsByID := make(map[string]n.ApplicationGatewayFrontendPort)
	for _, port := range *ports {
		portsByID[*port.ID] = port
	}

	listenersByID := make(map[listenerIdentifier]*n.ApplicationGatewayHTTPListener)
	// Update the listenerMap with the final listener lists
	for idx, listener := range *listeners {
		port, portExists := portsByID[*listener.FrontendPort.ID]

		listenerID := listenerIdentifier{
			UsePrivateIP: IsPrivateIPConfiguration(LookupIPConfigurationByID(c.appGw.FrontendIPConfigurations, listener.FrontendIPConfiguration.ID)),
		}

		if listener.Hostnames != nil && len(*listener.Hostnames) > 0 {
			listenerID.setHostNames(*listener.Hostnames)
		} else if listener.HostName != nil {
			listenerID.setHostNames([]string{*listener.HostName})
		}

		if portExists && port.Port != nil {
			listenerID.FrontendPort = Port(*port.Port)
		} else {
			glog.Errorf("Failed to find port '%s' referenced by listener '%s'", *listener.FrontendPort.ID, *listener.Name)
		}
		listenersByID[listenerID] = &((*listeners)[idx])
	}

	return listenersByID
}
