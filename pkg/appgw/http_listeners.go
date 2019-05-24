// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import "k8s.io/api/extensions/v1beta1"

func (builder *appGwConfigBuilder) Listeners(ingressList []*v1beta1.Ingress) (ConfigBuilder, error) {
	builder.appGwConfig.SslCertificates = builder.getSslCertificates(ingressList)
	builder.appGwConfig.FrontendPorts = builder.getFrontendPorts(ingressList)
	builder.appGwConfig.HTTPListeners, _ = builder.getFrontendListeners(ingressList)

	// App Gateway Rules can be configured to redirect HTTP traffic to HTTPS URLs.
	// In this step here we create the redirection configurations. These configs are attached to request routing rules
	// in the RequestRoutingRules step, which must be executed after Listeners.
	builder.appGwConfig.RedirectConfigurations = builder.getRedirectConfigurations(ingressList)

	return builder, nil
}
