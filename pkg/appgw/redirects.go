// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

// getRedirectConfigurations creates App Gateway redirect configuration based on Ingress annotations.
func (c *appGwConfigBuilder) getRedirectConfigurations(cbCtx *ConfigBuilderContext) *[]n.ApplicationGatewayRedirectConfiguration {
	if c.mem.redirectConfigs != nil {
		return c.mem.redirectConfigs
	}

	var redirectConfigs []n.ApplicationGatewayRedirectConfiguration

	// Iterate over all possible Listeners (generated from the K8s Ingress configurations)
	for listenerID, listenerConfig := range c.getListenerConfigs(cbCtx) {
		isHTTPS := listenerConfig.Protocol == n.HTTPS
		// What if multiple namespaces have a redirect configured?
		hasSslRedirect := listenerConfig.SslRedirectConfigurationName != ""

		// We will configure a Redirect only if the listener has TLS enabled (has a Certificate)
		if isHTTPS && hasSslRedirect {
			targetListener := resourceRef(c.appGwIdentifier.listenerID(generateListenerName(listenerID)))
			redirectConfigs = append(redirectConfigs, c.newSSLRedirectConfig(listenerConfig, targetListener))
			glog.V(5).Infof("Created redirection configuration %s for %s:%d; not yet linked to a routing rule", listenerConfig.SslRedirectConfigurationName, listenerID.HostName, listenerID.FrontendPort)
		}
	}

	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		er := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)

		// Listeners we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
		existingBlacklisted, existingNonBlacklisted := er.GetBlacklistedRedirects()

		brownfield.LogRedirects(existingBlacklisted, existingNonBlacklisted, redirectConfigs)

		// MergeRedirects would produce unique list of redirects based on Name. Blacklisted redirects,
		// which have the same name as a managed redirects would be overwritten.
		redirectConfigs = brownfield.MergeRedirects(existingBlacklisted, redirectConfigs)
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
