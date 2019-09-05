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
func (c *appGwConfigBuilder) getListeners(cbCtx *ConfigBuilderContext) *[]n.ApplicationGatewayHTTPListener {
	if c.mem.listeners != nil {
		return c.mem.listeners
	}
	var listeners []n.ApplicationGatewayHTTPListener

	if cbCtx.EnvVariables.EnableIstioIntegration {
		for listenerID, config := range c.getListenerConfigsFromIstio(cbCtx.IstioGateways, cbCtx.IstioVirtualServices) {
			listener := c.newListener(listenerID, config.Protocol)
			listeners = append(listeners, listener)
		}
	}

	for listenerID, config := range c.getListenerConfigs(cbCtx) {
		listener := c.newListener(listenerID, config.Protocol)
		if config.Protocol == n.HTTPS {
			sslCertificateID := c.appGwIdentifier.sslCertificateID(config.Secret.secretFullName())
			listener.SslCertificate = resourceRef(sslCertificateID)
		}
		listeners = append(listeners, listener)
		glog.V(5).Infof("Created listener %s with %s:%d", *listener.Name, listenerID.HostName, listenerID.FrontendPort)
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

	// Since getListeners() would be called multiple times within the life cycle of a Process(Event)
	// we cache the results of this function in what would be final place to store the Listeners.
	c.mem.listeners = &listeners
	return &listeners
}

// getListenerConfigs creates an intermediary representation of the listener configs based on the passed list of ingresses
func (c *appGwConfigBuilder) getListenerConfigs(cbCtx *ConfigBuilderContext) map[listenerIdentifier]listenerAzConfig {
	if c.mem.listenerConfigs != nil {
		return *c.mem.listenerConfigs
	}

	// TODO(draychev): Emit an error event if 2 namespaces define different TLS for the same domain!
	allListeners := make(map[listenerIdentifier]listenerAzConfig)
	for _, ingress := range cbCtx.IngressList {
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

func (c *appGwConfigBuilder) newListener(listenerID listenerIdentifier, protocol n.ApplicationGatewayProtocol) n.ApplicationGatewayHTTPListener {
	frontIPConfiguration := *LookupIPConfigurationByType(c.appGw.FrontendIPConfigurations, listenerID.UsePrivateIP)
	frontendPort := c.lookupFrontendPortByListenerIdentifier(listenerID)
	listenerName := generateListenerName(listenerID)
	return n.ApplicationGatewayHTTPListener{
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
}

func (c *appGwConfigBuilder) groupListenersByListenerIdentifier(listeners *[]n.ApplicationGatewayHTTPListener) map[listenerIdentifier]*n.ApplicationGatewayHTTPListener {
	listenersByID := make(map[listenerIdentifier]*n.ApplicationGatewayHTTPListener)
	// Update the listenerMap with the final listener lists
	for idx, listener := range *listeners {
		listenerID := listenerIdentifier{
			HostName:     *listener.HostName,
			FrontendPort: Port(*c.lookupFrontendPortByID(listener.FrontendPort.ID).Port),
			UsePrivateIP: IsPrivateIPConfiguration(LookupIPConfigurationByID(c.appGw.FrontendIPConfigurations, listener.FrontendIPConfiguration.ID)),
		}
		listenersByID[listenerID] = &((*listeners)[idx])
	}

	return listenersByID
}
