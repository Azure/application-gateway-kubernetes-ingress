// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

func (c *appGwConfigBuilder) getFrontendPorts(cbCtx *ConfigBuilderContext) *[]n.ApplicationGatewayFrontendPort {
	allPorts := make(map[int32]interface{})
	for _, ingress := range cbCtx.IngressList {
		fePorts, _ := c.processIngressRules(ingress)
		for port := range fePorts {
			allPorts[port] = nil
		}
	}

	// fallback to default listener as placeholder if no listener is available
	if len(allPorts) == 0 {
		port := defaultFrontendListenerIdentifier().FrontendPort
		allPorts[port] = nil
	}

	var frontendPorts []n.ApplicationGatewayFrontendPort
	for port := range allPorts {
		frontendPortName := generateFrontendPortName(port)
		frontendPorts = append(frontendPorts, n.ApplicationGatewayFrontendPort{
			Etag: to.StringPtr("*"),
			Name: &frontendPortName,
			ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
				Port: to.Int32Ptr(port),
			},
		})
	}
	sort.Sort(sorter.ByFrontendPortName(frontendPorts))
	return &frontendPorts
}
