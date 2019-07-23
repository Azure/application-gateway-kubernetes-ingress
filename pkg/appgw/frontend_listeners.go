// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"
	"strconv"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

// getListeners constructs the unique set of App Gateway HTTP listeners across all ingresses.
func (c *appGwConfigBuilder) getListeners(cbCtx *ConfigBuilderContext) *[]n.ApplicationGatewayHTTPListener {
	// TODO(draychev): this is for compatibility w/ RequestRoutingRules and should be removed ASAP
	var listeners []n.ApplicationGatewayHTTPListener

	if cbCtx.EnableIstioIntegration {
		for listenerID, config := range c.getListenerConfigsFromIstio(cbCtx.IstioGateways, cbCtx.IstioVirtualServices) {
			listener := c.newListener(listenerID, config.Protocol, cbCtx.EnvVariables)
			listeners = append(listeners, listener)
		}
	}

	for listenerID, config := range c.getListenerConfigs(cbCtx.IngressList) {
		listener := c.newListener(listenerID, config.Protocol, cbCtx.EnvVariables)
		if config.Protocol == n.HTTPS {
			sslCertificateID := c.appGwIdentifier.sslCertificateID(config.Secret.secretFullName())
			listener.SslCertificate = resourceRef(sslCertificateID)
		}
		listeners = append(listeners, listener)
	}

	if cbCtx.EnableBrownfieldDeployment {
		er := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)

		// Listeners we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
		existingBlacklisted, existingNonBlacklisted := er.GetBlacklistedListeners()

		brownfield.LogListeners(existingBlacklisted, existingNonBlacklisted, listeners)

		// MergeListeners would produce unique list of listeners based on Name. Blacklisted listeners,
		// which have the same name as a managed listeners would be overwritten.
		listeners = brownfield.MergeListeners(existingBlacklisted, listeners)
	}

	sort.Sort(sorter.ByListenerName(listeners))
	return &listeners
}

// getListenerConfigs creates an intermediary representation of the listener configs based on the passed list of ingresses
func (c *appGwConfigBuilder) getListenerConfigs(ingressList []*v1beta1.Ingress) map[listenerIdentifier]listenerAzConfig {
	// TODO(draychev): Emit an error event if 2 namespaces define different TLS for the same domain!
	allListeners := make(map[listenerIdentifier]listenerAzConfig)
	for _, ingress := range ingressList {
		glog.V(5).Infof("Processing Rules for Ingress: %s/%s", ingress.Namespace, ingress.Name)
		_, azListenerConfigs := c.processIngressRules(ingress)
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

	return allListeners
}

func (c *appGwConfigBuilder) newListener(listenerID listenerIdentifier, protocol n.ApplicationGatewayProtocol, envVariables environment.EnvVariables) n.ApplicationGatewayHTTPListener {
	frontendPortID := *c.lookupFrontendPortByListenerIdentifier(listenerID).ID
	listenerName := generateListenerName(listenerID)
	return n.ApplicationGatewayHTTPListener{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(listenerName),
		ID:   to.StringPtr(c.appGwIdentifier.listenerID(listenerName)),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			// TODO: expose this to external configuration
			FrontendIPConfiguration: resourceRef(*c.getIPConfigurationID(envVariables)),
			FrontendPort:            resourceRef(frontendPortID),
			Protocol:                protocol,
			HostName:                &listenerID.HostName,
		},
	}
}

func (c *appGwConfigBuilder) getIPConfigurationID(envVariables environment.EnvVariables) *string {
	usePrivateIP, _ := strconv.ParseBool(envVariables.UsePrivateIP)
	for _, ip := range *c.appGw.FrontendIPConfigurations {
		if ip.ApplicationGatewayFrontendIPConfigurationPropertiesFormat != nil &&
			((usePrivateIP && ip.PrivateIPAddress != nil) ||
				(!usePrivateIP && ip.PublicIPAddress != nil)) {
			return ip.ID
		}
	}

	// This should not happen as we are performing validation on frontIpConfiguration to make sure if have the required IP.
	return nil
}

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
				glog.Infof("[istio] AGIC does not support Gateway with Server.Port.Protocol=%+v", server.Port.Protocol)
				continue
			}
			for _, host := range server.Hosts {
				if _, exist := knownHosts[host]; !exist {
					continue
				}
				listenerID := listenerIdentifier{
					FrontendPort: int32(server.Port.Number),
					HostName:     host,
				}
				allListeners[listenerID] = listenerAzConfig{Protocol: n.HTTP}
			}
		}
	}

	// App Gateway must have at least one listener - the default one!
	if len(allListeners) == 0 {
		allListeners[defaultFrontendListenerIdentifier()] = listenerAzConfig{
			// Default protocol
			Protocol: n.HTTP,
		}
	}

	return allListeners
}

func (c *appGwConfigBuilder) groupListenersByListenerIdentifier(listeners *[]n.ApplicationGatewayHTTPListener) map[listenerIdentifier]*n.ApplicationGatewayHTTPListener {
	listenersByID := make(map[listenerIdentifier]*n.ApplicationGatewayHTTPListener)
	// Update the listenerMap with the final listener lists
	for idx, listener := range *listeners {
		port := c.lookupFrontendPortByID(listener.FrontendPort.ID)
		listenerID := listenerIdentifier{
			HostName:     *listener.HostName,
			FrontendPort: *port.Port,
		}
		listenersByID[listenerID] = &((*listeners)[idx])
	}

	return listenersByID
}
