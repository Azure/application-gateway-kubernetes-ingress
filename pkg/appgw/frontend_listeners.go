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
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

// getListeners constructs the unique set of App Gateway HTTP listeners across all ingresses.
func (c *appGwConfigBuilder) getListeners(kr *k8scontext.KubernetesResources) (*[]n.ApplicationGatewayHTTPListener, map[listenerIdentifier]*n.ApplicationGatewayHTTPListener) {
	// TODO(draychev): this is for compatibility w/ RequestRoutingRules and should be removed ASAP
	legacyMap := make(map[listenerIdentifier]*n.ApplicationGatewayHTTPListener)

	var listeners []n.ApplicationGatewayHTTPListener

	for listenerID, config := range c.getListenerConfigs(kr.IngressList) {
		listener := c.newListener(listenerID, config.Protocol, kr.EnvVariables)
		if config.Protocol == n.HTTPS {
			sslCertificateID := c.appGwIdentifier.sslCertificateID(config.Secret.secretFullName())
			listener.SslCertificate = resourceRef(sslCertificateID)
		}
		listeners = append(listeners, listener)
		legacyMap[listenerID] = &listener
	}

	// TODO(draychev): The second map we return is for compatibility w/ RequestRoutingRules and should be removed ASAP
	sort.Sort(sorter.ByListenerName(listeners))
	return &listeners, legacyMap
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

func (c *appGwConfigBuilder) newListener(listener listenerIdentifier, protocol n.ApplicationGatewayProtocol, envVariables environment.EnvVariables) n.ApplicationGatewayHTTPListener {
	frontendPortName := generateFrontendPortName(listener.FrontendPort)
	frontendPortID := c.appGwIdentifier.frontendPortID(frontendPortName)

	return n.ApplicationGatewayHTTPListener{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(generateListenerName(listener)),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			// TODO: expose this to external configuration
			FrontendIPConfiguration: resourceRef(*c.getIPConfigurationID(envVariables)),
			FrontendPort:            resourceRef(frontendPortID),
			Protocol:                protocol,
			HostName:                &listener.HostName,
		},
	}
}

func (c *appGwConfigBuilder) getIPConfigurationID(envVariables environment.EnvVariables) *string {
	usePrivateIP, _ := strconv.ParseBool(envVariables.UsePrivateIP)
	for _, ip := range *c.appGwConfig.FrontendIPConfigurations {
		if ip.ApplicationGatewayFrontendIPConfigurationPropertiesFormat != nil &&
			((usePrivateIP && ip.PrivateIPAddress != nil) ||
				(!usePrivateIP && ip.PublicIPAddress != nil)) {
			return ip.ID
		}
	}

	// This should not happen as we are performing validation on frontIpConfiguration to make sure if have the required IP.
	return nil
}
