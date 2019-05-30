package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"k8s.io/api/extensions/v1beta1"
)

// processIngressRules creates the sets of front end listeners and ports, and a map of azure config per listener for the given ingress.
func (builder *appGwConfigBuilder) processIngressRules(ingress *v1beta1.Ingress) (map[int32]interface{}, map[listenerIdentifier]listenerAzConfig) {
	frontendPorts := make(map[int32]interface{})

	ingressHostnameSecretIDMap := builder.newHostToSecretMap(ingress)
	azListenerConfigs := make(map[listenerIdentifier]listenerAzConfig)

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
			listenerIDHTTPS := generateListenerID(&rule, network.HTTPS, nil)
			frontendPorts[listenerIDHTTPS.FrontendPort] = nil

			felAzConfig := listenerAzConfig{
				Protocol:                     network.HTTPS,
				Secret:                       *secID,
				SslRedirectConfigurationName: generateSSLRedirectConfigurationName(ingress.Namespace, ingress.Name),
			}
			azListenerConfigs[listenerIDHTTPS] = felAzConfig

		}

		// Enable HTTP only if HTTPS is not configured OR if ingress annotated with 'ssl-redirect'
		sslRedirect, _ := annotations.IsSslRedirect(ingress)
		if sslRedirect || !httpsAvailable {
			listenerID := generateListenerID(&rule, network.HTTP, nil)
			frontendPorts[listenerID.FrontendPort] = nil
			felAzConfig := listenerAzConfig{
				Protocol: network.HTTP,
			}
			azListenerConfigs[listenerID] = felAzConfig
		}
	}
	return frontendPorts, azListenerConfigs
}
