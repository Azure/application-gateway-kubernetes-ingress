// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
)

func (c *appGwConfigBuilder) getListenersFromIngress(ingress *v1beta1.Ingress, env environment.EnvVariables) map[listenerIdentifier]listenerAzConfig {
	listeners := make(map[listenerIdentifier]listenerAzConfig)

	// if ingress has only backend configured
	if ingress.Spec.Backend != nil && len(ingress.Spec.Rules) == 0 {
		return listeners
	}

	// process ingress rules with TLS and Waf policy
	policy, _ := annotations.WAFPolicy(ingress)
	for ruleIdx := range ingress.Spec.Rules {
		rule := &ingress.Spec.Rules[ruleIdx]
		if rule.HTTP == nil {
			continue
		}
		_, ruleListeners := c.processIngressRuleWithTLS(rule, ingress, env)

		applyToListener := false
		if policy != "" {
			applyToListener = c.applyToListener(rule)
		}

		for k, v := range ruleListeners {
			if applyToListener {
				glog.V(3).Infof("Attach WAF policy: %s to listener: %s", policy, generateListenerName(k))
				v.FirewallPolicy = policy
			}
			listeners[k] = v
		}
	}

	return listeners
}

func (c *appGwConfigBuilder) applyToListener(rule *v1beta1.IngressRule) bool {
	for pathIdx := range rule.HTTP.Paths {
		path := &rule.HTTP.Paths[pathIdx]
		// if there is path that is /, /* , empty string, then apply the waf policy to the listener.
		if len(path.Path) == 0 || path.Path == "/" || path.Path == "/*" {
			return true
		}
	}
	return false
}

func (c *appGwConfigBuilder) processIngressRuleWithTLS(rule *v1beta1.IngressRule, ingress *v1beta1.Ingress, env environment.EnvVariables) (map[Port]interface{}, map[listenerIdentifier]listenerAzConfig) {
	frontendPorts := make(map[Port]interface{})

	// certificate from ingress TLS spec
	ingressHostnameSecretIDMap := c.newHostToSecretMap(ingress)

	listeners := make(map[listenerIdentifier]listenerAzConfig)

	// Private IP is used when either annotation use-private-ip or USE_PRIVATE_IP env variable is true.
	usePrivateIPFromAnnotation, _ := annotations.UsePrivateIP(ingress)
	usePrivateIPForIngress := usePrivateIPFromAnnotation || env.UsePrivateIP == "true"

	appgwCertName, _ := annotations.GetAppGwSslCertificate(ingress)
	if len(appgwCertName) > 0 {
		// logging to see the namespace of the ingress annotated with appgw-ssl-certificate
		glog.V(5).Infof("Found annotation appgw-ssl-certificate: %s in ingress %s/%s", appgwCertName, ingress.Namespace, ingress.Name)
	}

	cert, secID := c.getCertificate(ingress, rule.Host, ingressHostnameSecretIDMap)
	hasTLS := (cert != nil || len(appgwCertName) > 0)

	sslRedirect, _ := annotations.IsSslRedirect(ingress)

	// If a certificate is available we enable only HTTPS; unless ingress is annotated with ssl-redirect - then
	// we enable HTTPS as well as HTTP, and redirect HTTP to HTTPS;
	if hasTLS {
		listenerID := generateListenerID(ingress, rule, n.HTTPS, nil, usePrivateIPForIngress)
		frontendPorts[Port(listenerID.FrontendPort)] = nil
		// Only associate the Listener with a Redirect if redirect is enabled
		redirect := ""
		if sslRedirect {
			redirect = generateSSLRedirectConfigurationName(listenerID)
		}

		azConf := listenerAzConfig{
			Protocol:                     n.HTTPS,
			SslRedirectConfigurationName: redirect,
		}
		// appgw-ssl-certificate annotation will be ignored if TLS spec found
		if cert != nil {
			azConf.Secret = *secID

		} else if len(appgwCertName) > 0 {
			// the cert annotated can be referred across namespace,
			// set namespace to "" to ignore namespace
			azConf.Secret = secretIdentifier{
				Name:      appgwCertName,
				Namespace: "",
			}
		}

		listeners[listenerID] = azConf
	}
	// Enable HTTP only if HTTPS is not configured OR if ingress annotated with 'ssl-redirect'
	if sslRedirect || !hasTLS {
		listenerID := generateListenerID(ingress, rule, n.HTTP, nil, usePrivateIPForIngress)
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
