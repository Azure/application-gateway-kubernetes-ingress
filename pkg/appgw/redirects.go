package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) getRedirectConfigurations(ingressList []*v1beta1.Ingress) *[]n.ApplicationGatewayRedirectConfiguration {
	var redirectConfigs []n.ApplicationGatewayRedirectConfiguration

	for listenerID, config := range builder.getListenerConfigs(ingressList) {
		isHTTPS := config.Protocol == n.HTTPS
		hasSslRedirect := config.SslRedirectConfigurationName != ""

		if !isHTTPS || !hasSslRedirect {
			continue
		}

		targetListener := resourceRef(builder.appGwIdentifier.httpListenerID(generateHTTPListenerName(listenerID)))
		redirectConfigs = append(redirectConfigs, newSSLRedirectConfig(config, targetListener))
	}

	return &redirectConfigs
}

func newSSLRedirectConfig(azureConfig frontendListenerAzureConfig, targetListener *n.SubResource) n.ApplicationGatewayRedirectConfiguration {
	props := n.ApplicationGatewayRedirectConfigurationPropertiesFormat{
		RedirectType:       n.Permanent,
		TargetListener:     targetListener,
		IncludePath:        to.BoolPtr(true),
		IncludeQueryString: to.BoolPtr(true),
	}

	return n.ApplicationGatewayRedirectConfiguration{
		Etag: to.StringPtr("*"),
		Name: &azureConfig.SslRedirectConfigurationName,
		ApplicationGatewayRedirectConfigurationPropertiesFormat: &props,
	}
}
