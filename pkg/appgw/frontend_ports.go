// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

func (c *appGwConfigBuilder) getFrontendPorts(cbCtx *ConfigBuilderContext) *[]n.ApplicationGatewayFrontendPort {
	if c.mem.ports != nil {
		return c.mem.ports
	}

	allPorts := make(map[Port]interface{})

	if cbCtx.EnvVariables.EnableIstioIntegration {
		for _, gwy := range cbCtx.IstioGateways {
			for _, server := range gwy.Spec.Servers {
				allPorts[Port(server.Port.Number)] = nil
			}
		}
	}

	for _, ingress := range cbCtx.IngressList {
		for port := range c.getFrontendPortsFromIngress(ingress, cbCtx.EnvVariables) {
			allPorts[port] = nil
		}
	}

	// fallback to default listener as placeholder if no listener is available
	if len(allPorts) == 0 {
		port := defaultFrontendListenerIdentifier().FrontendPort
		allPorts[port] = nil
	}

	fmt.Println(">>>>>>>>>>>>>>>>>>>>> PORTS:", allPorts)

	var frontendPorts []n.ApplicationGatewayFrontendPort
	for port := range allPorts {
		frontendPortName := generateFrontendPortName(port)
		frontendPorts = append(frontendPorts, n.ApplicationGatewayFrontendPort{
			Etag: to.StringPtr("*"),
			Name: &frontendPortName,
			ID:   to.StringPtr(c.appGwIdentifier.frontendPortID(frontendPortName)),
			ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
				Port: to.Int32Ptr(int32(port)),
			},
		})
	}

	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		er := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)

		// Ports we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
		existingBlacklisted, existingNonBlacklisted := er.GetBlacklistedPorts()

		brownfield.LogPorts(existingBlacklisted, existingNonBlacklisted, frontendPorts)

		// MergePorts would produce unique list of ports based on Name. Blacklisted ports,
		// which have the same name as a managed ports would be overwritten.
		frontendPorts = brownfield.MergePorts(existingBlacklisted, frontendPorts)
	}

	sort.Sort(sorter.ByFrontendPortName(frontendPorts))
	c.mem.ports = &frontendPorts
	return &frontendPorts
}
