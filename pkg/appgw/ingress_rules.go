// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
)

func (c *appGwConfigBuilder) getFrontendPortsFromIngress(ingress *v1beta1.Ingress, env environment.EnvVariables) map[Port]interface{} {
	frontendPorts := make(map[Port]interface{})
	for ruleIdx := range ingress.Spec.Rules {
		rule := &ingress.Spec.Rules[ruleIdx]
		if rule.HTTP == nil {
			continue
		}

		ruleFrontendPorts, _ := c.processIngressRule(rule, ingress, env)
		for port, _ := range ruleFrontendPorts {
			frontendPorts[port] = nil
		}
	}
	return frontendPorts
}

func (c *appGwConfigBuilder) getListenersFromIngress(ingress *v1beta1.Ingress, env environment.EnvVariables) map[listenerIdentifier]listenerAzConfig {
	listeners := make(map[listenerIdentifier]listenerAzConfig)
	for ruleIdx := range ingress.Spec.Rules {
		rule := &ingress.Spec.Rules[ruleIdx]
		if rule.HTTP == nil {
			continue
		}

		_, ruleListeners := c.processIngressRule(rule, ingress, env)
		for k, v := range ruleListeners {
			listeners[k] = v
		}
	}
	return listeners
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
		frontendPorts[Port(listenerID.FrontendPort)] = nil
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
		frontendPorts[Port(listenerID.FrontendPort)] = nil
		listeners[listenerID] = listenerAzConfig{
			Protocol: n.HTTP,
		}
	}
	return frontendPorts, listeners
}

func (c *appGwConfigBuilder) newBackendIdsFiltered(cbCtx *ConfigBuilderContext) map[backendIdentifier]interface{} {
	if c.mem.backendIDs != nil {
		return *c.mem.backendIDs
	}

	backendIDs := make(map[backendIdentifier]interface{})
	for _, ingress := range cbCtx.IngressList {
		if ingress.Spec.Backend != nil {
			backendID := generateBackendID(ingress, nil, nil, ingress.Spec.Backend)
			glog.V(3).Info("Found default backend:", backendID.serviceKey())
			backendIDs[backendID] = nil
		}
		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			if rule.HTTP == nil {
				// skip no http rule
				glog.V(5).Infof("[%s] Skip rule #%d for host '%s' - it has no HTTP rules.", ingress.Namespace, ruleIdx+1, rule.Host)
				continue
			}
			for pathIdx := range rule.HTTP.Paths {
				path := &rule.HTTP.Paths[pathIdx]
				backendID := generateBackendID(ingress, rule, path, &path.Backend)
				glog.V(5).Info("Found backend:", backendID.serviceKey())
				backendIDs[backendID] = nil
			}
		}
	}

	finalBackendIDs := make(map[backendIdentifier]interface{})
	serviceSet := newServiceSet(&cbCtx.ServiceList)
	// Filter out backends, where Ingresses reference non-existent Services
	for be := range backendIDs {
		if _, exists := serviceSet[be.serviceKey()]; !exists {
			glog.Errorf("Ingress %s/%s references non existent Service %s. Please correct the Service section of your Kubernetes YAML", be.Ingress.Namespace, be.Ingress.Name, be.serviceKey())
			// TODO(draychev): Enable this filter when we are certain this won't break anything!
			// continue
		}
		finalBackendIDs[be] = nil
	}

	c.mem.backendIDs = &finalBackendIDs
	return finalBackendIDs
}
