// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

// getRedirectConfigurations creates App Gateway redirect configuration based on Ingress annotations.
func (c *appGwConfigBuilder) getRedirectConfigurations(cbCtx *ConfigBuilderContext) *[]n.ApplicationGatewayRedirectConfiguration {
	if c.mem.redirectConfigs != nil {
		return c.mem.redirectConfigs
	}

	redirectConfigs := []n.ApplicationGatewayRedirectConfiguration{}

	// Iterate over all possible Listeners (generated from the K8s Ingress configurations)
	httpListenersMap := c.groupListenersByListenerIdentifier(cbCtx)
	for listenerID, listenerConfig := range c.getListenerConfigs(cbCtx) {
		httpListener, exists := httpListenersMap[listenerID]
		if !exists {
			klog.Errorf("Redirect will not be created for target listener %+v as listener does not exist in listenerMap", listenerID)
			continue
		}

		isHTTPS := listenerConfig.Protocol == n.HTTPS
		// What if multiple namespaces have a redirect configured?
		hasSslRedirect := listenerConfig.SslRedirectConfigurationName != ""

		// We will configure a Redirect only if the listener has TLS enabled (has a Certificate)
		if isHTTPS && hasSslRedirect {
			targetListener := resourceRef(*httpListener.ID)
			redirectConfigs = append(redirectConfigs, c.newSSLRedirectConfig(listenerConfig, targetListener))
			klog.Infof("Created redirection configuration %s for (%s,%d); not yet linked to a routing rule", listenerConfig.SslRedirectConfigurationName, listenerID.HostNames, listenerID.FrontendPort)
		}
	}

	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		er := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, cbCtx.AllowedTargets, nil)

		// Listeners we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
		var existingNonAllowed []n.ApplicationGatewayRedirectConfiguration
		var existingAllowed []n.ApplicationGatewayRedirectConfiguration

		if cbCtx.EnvVariables.UseAllowedTargetsBrownfieldDeployment {
			existingNonAllowed, existingAllowed = er.GetNotWhitelistedRedirects()
		} else {
			existingNonAllowed, existingAllowed = er.GetBlacklistedRedirects()
		}

		brownfield.LogRedirects(existingNonAllowed, existingAllowed, redirectConfigs)

		// MergeRedirects would produce unique list of redirects based on Name. Blacklisted redirects,
		// which have the same name as a managed redirects would be overwritten.
		redirectConfigs = brownfield.MergeRedirects(existingNonAllowed, redirectConfigs)
	}

	sort.Sort(sorter.ByRedirectName(redirectConfigs))
	c.mem.redirectConfigs = &redirectConfigs
	return &redirectConfigs
}

// newSSLRedirectConfig creates a new Redirect in the form of a ApplicationGatewayRedirectConfiguration struct.
func (c *appGwConfigBuilder) newSSLRedirectConfig(listenerConfig listenerAzConfig, targetListener *n.SubResource) n.ApplicationGatewayRedirectConfiguration {
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
		ID:   to.StringPtr(c.appGwIdentifier.redirectConfigurationID(listenerConfig.SslRedirectConfigurationName)),
		ApplicationGatewayRedirectConfigurationPropertiesFormat: &props,
	}
}

func (c *appGwConfigBuilder) groupRedirectsByID(redirects *[]n.ApplicationGatewayRedirectConfiguration) *map[string]interface{} {
	redirectsSet := make(map[string]interface{})
	for _, redirect := range *redirects {
		redirectsSet[*redirect.ID] = nil
	}
	return &redirectsSet
}

func (c *appGwConfigBuilder) getSslRedirectConfigResourceReference(targetListener listenerIdentifier) *n.SubResource {
	configName := generateSSLRedirectConfigurationName(targetListener)
	sslRedirectConfigID := c.appGwIdentifier.redirectConfigurationID(configName)
	return resourceRef(sslRedirectConfigID)
}
