package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"k8s.io/api/extensions/v1beta1"
)

// processIngressRules creates the sets of front end listeners and ports, and a map of azure config per listener for the given ingress.
func (builder *appGwConfigBuilder) processIngressRules(ingress *v1beta1.Ingress) (utils.UnorderedSet, map[frontendListenerIdentifier]*frontendListenerAzureConfig) {
	frontendPorts := utils.NewUnorderedSet()

	ingressHostnameSecretIDMap := builder.newHostToSecretMap(ingress)
	azListenerConfigs := make(map[frontendListenerIdentifier]*frontendListenerAzureConfig)

	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			// skip no http rule
			continue
		}

		cert, secID := builder.getCertificate(ingress, rule.Host, ingressHostnameSecretIDMap)
		httpsAvailable := cert != nil

		// If a cert is a available it is implied that we should enable only HTTPS.
		// TODO: Once we introduce an `ssl-redirect` annotation we should enable HTTP for HTTPS rules as well, with the correct SSL redirect configurations setup.
		if httpsAvailable {
			listenerIDHTTPS := generateFrontendListenerID(&rule, network.HTTPS, nil)
			frontendPorts.Insert(listenerIDHTTPS.FrontendPort)

			felAzConfig := &frontendListenerAzureConfig{
				Protocol:                     network.HTTPS,
				Secret:                       *secID,
				SslRedirectConfigurationName: generateSSLRedirectConfigurationName(ingress.Namespace, ingress.Name),
			}
			azListenerConfigs[listenerIDHTTPS] = felAzConfig

		}

		if annotations.IsSslRedirect(ingress) || !httpsAvailable {
			// Enable HTTP only if HTTPS has not been specified or if an SSL-redirect annotation has been set to `true`.
			listenerIDHTTP := generateFrontendListenerID(&rule, network.HTTP, nil)
			frontendPorts.Insert(listenerIDHTTP.FrontendPort)
			felAzConfig := &frontendListenerAzureConfig{
				Protocol: network.HTTP,
			}
			azListenerConfigs[listenerIDHTTP] = felAzConfig
		}
	}
	return frontendPorts, azListenerConfigs
}
