package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"k8s.io/api/extensions/v1beta1"
)

// processIngressRules creates the sets of front end listeners and ports, and a map of azure config per listener for the given ingress.
func (c *appGwConfigBuilder) processIngressRules(ingress *v1beta1.Ingress) (map[int32]interface{}, map[listenerIdentifier]listenerAzConfig) {
	frontendPorts := make(map[int32]interface{})

	ingressHostnameSecretIDMap := c.newHostToSecretMap(ingress)
	listeners := make(map[listenerIdentifier]listenerAzConfig)

	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		cert, secID := c.getCertificate(ingress, rule.Host, ingressHostnameSecretIDMap)
		hasTLS := cert != nil
		sslRedirect, _ := annotations.IsSslRedirect(ingress)
		// If a certificate is available we enable only HTTPS; unless ingress is annotated with ssl-redirect - then
		// we enable HTTPS as well as HTTP, and redirect HTTP to HTTPS.
		if hasTLS {
			listenerID := generateListenerID(&rule, network.HTTPS, nil)
			frontendPorts[listenerID.FrontendPort] = nil
			// Only associate the Listener with a Redirect if redirect is enabled
			redirect := ""
			if sslRedirect {
				redirect = generateSSLRedirectConfigurationName(ingress.Namespace, ingress.Name)
			}

			listeners[listenerID] = listenerAzConfig{
				Protocol:                     network.HTTPS,
				Secret:                       *secID,
				SslRedirectConfigurationName: redirect,
			}
		}

		// Enable HTTP only if HTTPS is not configured OR if ingress annotated with 'ssl-redirect'
		if sslRedirect || !hasTLS {
			listenerID := generateListenerID(&rule, network.HTTP, nil)
			frontendPorts[listenerID.FrontendPort] = nil
			listeners[listenerID] = listenerAzConfig{
				Protocol: network.HTTP,
			}
		}

	}
	return frontendPorts, listeners
}
