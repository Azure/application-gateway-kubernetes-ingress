package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"k8s.io/api/extensions/v1beta1"
)

// processIngressRules creates the sets of front end listeners and ports, and a map of azure config per listener for the given ingress.
func (builder *appGwConfigBuilder) processIngressRules(ingress *v1beta1.Ingress) (utils.UnorderedSet, map[frontendListenerIdentifier]frontendListenerAzureConfig) {
	frontendPorts := utils.NewUnorderedSet()

	ingressHostnameSecretIDMap := builder.newHostToSecretMap(ingress)
	azListenerConfigs := make(map[frontendListenerIdentifier]frontendListenerAzureConfig)

	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			// skip no http rule
			continue
		}

		cert, secID := builder.getCertificate(ingress, rule.Host, ingressHostnameSecretIDMap)
		httpsAvailable := cert != nil

		// If a certificate is available we enable only HTTPS; unless ingress is annotated with ssl-redirect - then
		// we enable HTTPS as well as HTTP, and redirect HTTP to HTTPS.
		if httpsAvailable {
			listenerIDHTTPS := generateFrontendListenerID(&rule, network.HTTPS, nil)
			frontendPorts.Insert(listenerIDHTTPS.FrontendPort)

			felAzConfig := frontendListenerAzureConfig{
				Protocol:                     network.HTTPS,
				Secret:                       *secID,
				SslRedirectConfigurationName: generateSSLRedirectConfigurationName(ingress.Namespace, ingress.Name),
			}
			azListenerConfigs[listenerIDHTTPS] = felAzConfig

		}

		// Enable HTTP only if HTTPS is not configured OR if ingress annotated with 'ssl-redirect'
		if annotations.IsSslRedirect(ingress) || !httpsAvailable {
			listenerIDHTTP := generateFrontendListenerID(&rule, network.HTTP, nil)
			frontendPorts.Insert(listenerIDHTTP.FrontendPort)
			felAzConfig := frontendListenerAzureConfig{
				Protocol: network.HTTP,
			}
			azListenerConfigs[listenerIDHTTP] = felAzConfig
		}
	}
	return frontendPorts, azListenerConfigs
}
