package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
)

// getRedirectConfigurations creates App Gateway redirect configuration based on Ingress annotations.
func (builder *appGwConfigBuilder) getRedirectConfigurations(ingressList []*v1beta1.Ingress) *[]n.ApplicationGatewayRedirectConfiguration {
	var redirectConfigs []n.ApplicationGatewayRedirectConfiguration

	// Iterate over all possible Listeners (generated from the K8s Ingress configurations)
	for listenerID, listenerConfig := range builder.getListenerConfigs(ingressList) {
		isHTTPS := listenerConfig.Protocol == n.HTTPS
		hasSslRedirect := listenerConfig.SslRedirectConfigurationName != ""

		// We will configure a Redirect only if the listener has TLS enabled (has a Certificate)
		if isHTTPS && hasSslRedirect {
			targetListener := resourceRef(builder.appGwIdentifier.listenerID(generateListenerName(listenerID)))
			newRedirect := newSSLRedirectConfig(listenerConfig, targetListener)
			redirectConfigs = append(redirectConfigs, newRedirect)
			redirectJSON, _ := newRedirect.MarshalJSON()
			glog.Infof("Created redirection configuration; not attached to a routing rule yet. Configuration: %s", redirectJSON)
		}
	}

	return &redirectConfigs
}

// newSSLRedirectConfig creates a new Redirect in the form of a ApplicationGatewayRedirectConfiguration struct.
func newSSLRedirectConfig(listenerConfig listenerAzConfig, targetListener *n.SubResource) n.ApplicationGatewayRedirectConfiguration {
	props := n.ApplicationGatewayRedirectConfigurationPropertiesFormat{
		// RedirectType could be one of: 301/Permanent, 302/Found, 303/See Other, 307/Temporary
		RedirectType: n.Permanent,

		// To what listener we are redirecting.
		TargetListener: targetListener,

		// Include the path in the redirected URL.
		IncludePath: to.BoolPtr(true),

		// Include the query string in the redirected URL.
		IncludeQueryString: to.BoolPtr(true),
	}

	// To create a new redirect we need a
	return n.ApplicationGatewayRedirectConfiguration{
		Etag: to.StringPtr("*"),
		Name: &listenerConfig.SslRedirectConfigurationName,
		ApplicationGatewayRedirectConfigurationPropertiesFormat: &props,
	}
}
