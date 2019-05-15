package appgw

import "k8s.io/api/extensions/v1beta1"

func (builder *appGwConfigBuilder) getListenerConfigs(ingressList []*v1beta1.Ingress) map[frontendListenerIdentifier]*frontendListenerAzureConfig {
	httpListenersAzureConfigMap := make(map[frontendListenerIdentifier]*frontendListenerAzureConfig)

 	for _, ingress := range ingressList {
		_, _, configMap := builder.processIngressRules(ingress)
		for k, v := range configMap {
			httpListenersAzureConfigMap[k] = v
		}
	}

 	return httpListenersAzureConfigMap
}
