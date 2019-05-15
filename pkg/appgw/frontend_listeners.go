package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) getFrontendListeners(ingressList []*v1beta1.Ingress) (*[]network.ApplicationGatewayHTTPListener, map[frontendListenerIdentifier]*network.ApplicationGatewayHTTPListener) {
	// TODO(draychev): this is for compatibility w/ RequestRoutingRules and should be removed ASAP
	legacyMap := make(map[frontendListenerIdentifier]*network.ApplicationGatewayHTTPListener)

	listenerAzureConfigs := builder.getListenerConfigs(ingressList)
	var httpListeners []network.ApplicationGatewayHTTPListener

	for listener := range builder.getFrontendListenersMap(ingressList) {
		var secretFullName string
		var protocol network.ApplicationGatewayProtocol

		if config := listenerAzureConfigs[listener]; config != nil {
			protocol = config.Protocol
			secretFullName = config.Secret.secretFullName()
		} else {
			// Default protocol
			protocol = network.HTTP
		}

		httpListener := builder.newListener(listener, protocol)

		listenerHasHostname := len(*httpListener.ApplicationGatewayHTTPListenerPropertiesFormat.HostName) > 0

		if protocol == network.HTTPS {
			sslCertificateName := secretFullName
			sslCertificateID := builder.appGwIdentifier.sslCertificateID(sslCertificateName)
			httpListener.SslCertificate = resourceRef(sslCertificateID)

			if listenerHasHostname {
				httpListener.RequireServerNameIndication = to.BoolPtr(true)
			}
		}

		if listenerHasHostname {
			// Put the listener at the front of the list!
			httpListeners = append([]network.ApplicationGatewayHTTPListener{httpListener}, httpListeners...)
		} else {
			httpListeners = append(httpListeners, httpListener)
		}

		legacyMap[listener] = &httpListener
	}
	// TODO(draychev): The second parameter is for compatibility w/ RequestRoutingRules and should be removed ASAP
	return &httpListeners, legacyMap
}

func (builder *appGwConfigBuilder) getFrontendListenersMap(ingressList []*v1beta1.Ingress) map[frontendListenerIdentifier]interface{} {
	allListeners := make(map[frontendListenerIdentifier]interface{})
	for _, ingress := range ingressList {
		feListeners, _, _ := builder.processIngressRules(ingress)
		for _, listener := range feListeners.ToSlice() {
			l := listener.(frontendListenerIdentifier)
			allListeners[l] = nil
		}
	}

	if len(allListeners) == 0 {
		dflt := defaultFrontendListenerIdentifier()
		allListeners[dflt] = nil
	}

	return allListeners
}

// TODO(draychev): This function is used in a few places that require UnorderedSet type
func (builder *appGwConfigBuilder) getFrontendListenersSet(ingressList []*v1beta1.Ingress) utils.UnorderedSet {
	frontendListeners := utils.NewUnorderedSet()
	for listener := range builder.getFrontendListenersMap(ingressList) {
		frontendListeners.Insert(listener)
	}
	return frontendListeners
}

// getListenerConfigs iterates over all ingresses given and collects unique frontend listeners azure configs
func (builder *appGwConfigBuilder) getListenerConfigs(ingressList []*v1beta1.Ingress) map[frontendListenerIdentifier]*frontendListenerAzureConfig {
	httpListenersAzureConfigMap := make(map[frontendListenerIdentifier]*frontendListenerAzureConfig)

	for _, ingress := range ingressList {
		_, _, azListenerConfigs := builder.processIngressRules(ingress)
		for felIdentifier, felAzConfig := range azListenerConfigs {
			httpListenersAzureConfigMap[felIdentifier] = felAzConfig
		}
	}

	return httpListenersAzureConfigMap
}

func (builder *appGwConfigBuilder) newListener(listener frontendListenerIdentifier, protocol network.ApplicationGatewayProtocol) network.ApplicationGatewayHTTPListener {
	frontendPortName := generateFrontendPortName(listener.FrontendPort)
	frontendPortID := builder.appGwIdentifier.frontendPortID(frontendPortName)

	feConfigs := *builder.appGwConfig.FrontendIPConfigurations
	firstConfig := feConfigs[0]

	return network.ApplicationGatewayHTTPListener{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(generateHTTPListenerName(listener)),
		ApplicationGatewayHTTPListenerPropertiesFormat: &network.ApplicationGatewayHTTPListenerPropertiesFormat{
			// TODO: expose this to external configuration
			FrontendIPConfiguration: resourceRef(*firstConfig.ID),
			FrontendPort:            resourceRef(frontendPortID),
			Protocol:                protocol,
			HostName:                &listener.HostName,
		},
	}
}
