// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"k8s.io/api/extensions/v1beta1"
)

func (c *appGwConfigBuilder) Listeners(ingressList []*v1beta1.Ingress) error {
	c.appGwConfig.SslCertificates = c.getSslCertificates(ingressList)
	c.appGwConfig.FrontendPorts = c.getFrontendPorts(ingressList)
	c.appGwConfig.HTTPListeners, _ = c.getListeners(ingressList)

	// App Gateway Rules can be configured to redirect HTTP traffic to HTTPS URLs.
	// In this step here we create the redirection configurations. These configs are attached to request routing rules
	// in the RequestRoutingRules step, which must be executed after Listeners.
	c.appGwConfig.RedirectConfigurations = c.getRedirectConfigurations(ingressList)

	return nil
}
