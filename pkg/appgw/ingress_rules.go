// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
)

// processIngressRules creates the sets of front end listeners and ports, and a map of azure config per listener for the given ingress.
func (c *appGwConfigBuilder) processIngressRules(ingress *v1beta1.Ingress, env environment.EnvVariables) (map[Port]interface{}, map[listenerIdentifier]listenerAzConfig) {
	frontendPorts := make(map[Port]interface{})
	listeners := make(map[listenerIdentifier]listenerAzConfig)
	for ruleIdx := range ingress.Spec.Rules {
		rule := &ingress.Spec.Rules[ruleIdx]
		if rule.HTTP == nil {
			continue
		}

		ruleFrontendPorts, ruleListeners := c.processIngressRule(rule, ingress, env)
		for port, _ := range ruleFrontendPorts {
			frontendPorts[port] = nil
		}
		for listener, listenerConfig := range ruleListeners {
			listeners[listener] = listenerConfig
		}
	}
	return frontendPorts, listeners
}

func (c *appGwConfigBuilder) processIngressRule(rule *v1beta1.IngressRule, ingress *v1beta1.Ingress, env environment.EnvVariables) (map[Port]interface{}, map[listenerIdentifier]listenerAzConfig) {
	frontendPorts := make(map[Port]interface{})
	ingressHostnameSecretIDMap := c.newHostToSecretMap(ingress)
	listeners := make(map[listenerIdentifier]listenerAzConfig)

	// Private IP is used when either annotation use-private-ip or USE_PRIVATE_IP env variable is true.
	usePrivateIPFromAnnotation, _ := annotations.UsePrivateIP(ingress)
	usePrivateIPForIngress := usePrivateIPFromAnnotation || env.UsePrivateIP == "true"

	cert, secID := c.getCertificate(ingress, rule.Host, ingressHostnameSecretIDMap)
	hasTLS := cert != nil
	sslRedirect, _ := annotations.IsSslRedirect(ingress)
	// If a certificate is available we enable only HTTPS; unless ingress is annotated with ssl-redirect - then
	// we enable HTTPS as well as HTTP, and redirect HTTP to HTTPS.
	if hasTLS {
		listenerID := generateListenerID(rule, n.HTTPS, nil, usePrivateIPForIngress)
		frontendPorts[listenerID.FrontendPort] = nil
		// Only associate the Listener with a Redirect if redirect is enabled
		redirect := ""
		if sslRedirect {
			redirect = generateSSLRedirectConfigurationName(listenerID)
		}

		listeners[listenerID] = listenerAzConfig{
			Protocol:                     n.HTTPS,
			Secret:                       *secID,
			SslRedirectConfigurationName: redirect,
		}
	}

	// Enable HTTP only if HTTPS is not configured OR if ingress annotated with 'ssl-redirect'
	if sslRedirect || !hasTLS {
		listenerID := generateListenerID(rule, n.HTTP, nil, usePrivateIPForIngress)
		frontendPorts[listenerID.FrontendPort] = nil
		listeners[listenerID] = listenerAzConfig{
			Protocol: n.HTTP,
		}
	}

	return frontendPorts, listeners
}
