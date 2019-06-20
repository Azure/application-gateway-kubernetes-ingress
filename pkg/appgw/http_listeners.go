// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

func (c *appGwConfigBuilder) Listeners(cbCtx *ConfigBuilderContext) error {

	c.appGwConfig.SslCertificates = c.getSslCertificates(cbCtx.IngressList)
	c.appGwConfig.FrontendPorts = c.getFrontendPorts(cbCtx.IngressList)
	c.appGwConfig.HTTPListeners, _ = c.getListeners(cbCtx)

	// App Gateway Rules can be configured to redirect HTTP traffic to HTTPS URLs.
	// In this step here we create the redirection configurations. These configs are attached to request routing rules
	// in the RequestRoutingRules step, which must be executed after Listeners.
	c.appGwConfig.RedirectConfigurations = c.getRedirectConfigurations(cbCtx.IngressList)

	return nil
}
