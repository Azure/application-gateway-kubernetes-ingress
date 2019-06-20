// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

func (c *appGwConfigBuilder) Listeners(kr *ConfigBuilderContext) error {

	c.appGwConfig.SslCertificates = c.getSslCertificates(kr.IngressList)
	c.appGwConfig.FrontendPorts = c.getFrontendPorts(kr.IngressList)
	c.appGwConfig.HTTPListeners, _ = c.getListeners(kr)

	// App Gateway Rules can be configured to redirect HTTP traffic to HTTPS URLs.
	// In this step here we create the redirection configurations. These configs are attached to request routing rules
	// in the RequestRoutingRules step, which must be executed after Listeners.
	c.appGwConfig.RedirectConfigurations = c.getRedirectConfigurations(kr.IngressList)

	return nil
}
