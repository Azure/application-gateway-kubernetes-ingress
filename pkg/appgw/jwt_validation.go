// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
)

// buildEntraJWTMergePayload upserts Helm JWT configs onto existing Azure configs and records rule bindings.
func (c *appGwConfigBuilder) buildEntraJWTMergePayload(cbCtx *ConfigBuilderContext, ruleBindings map[string]string) *azure.JWTMergePayload {
	payload := azure.EmptyJWTMerge()

	var helmConfigs []azure.EntraJWTValidationConfig
	for _, cfg := range cbCtx.EnvVariables.EntraJWTConfigs {
		helmConfigs = append(helmConfigs, azure.NewHelmEntraJWTConfig(
			c.appGwIdentifier.SubscriptionID,
			c.appGwIdentifier.ResourceGroup,
			c.appGwIdentifier.AppGwName,
			cfg.Name,
			cfg.TenantID,
			cfg.ClientID,
			cfg.UnauthorizedAction,
			cfg.Audiences,
		))
	}

	payload.Configs = azure.MergeEntraJWTConfigs(cbCtx.ExistingEntraJWTConfigs, helmConfigs)
	if ruleBindings != nil {
		payload.RuleBindings = ruleBindings
	}
	return payload
}

// resolveListenerJWTBindings maps each HTTPS listener to an Entra JWT config name from Ingress annotations.
func (c *appGwConfigBuilder) resolveListenerJWTBindings(cbCtx *ConfigBuilderContext) map[listenerIdentifier]string {
	bindings := map[listenerIdentifier]string{}
	conflicts := map[listenerIdentifier]bool{}
	available := azure.ConfigNames(cbCtx.ExistingEntraJWTConfigs)
	for _, cfg := range cbCtx.EnvVariables.EntraJWTConfigs {
		available[cfg.Name] = struct{}{}
	}
	listenerConfigs := c.getListenerConfigs(cbCtx)

	for _, ingress := range cbCtx.IngressList {
		jwtName, err := annotations.EntraJWTConfigName(ingress)
		if err != nil || strings.TrimSpace(jwtName) == "" {
			continue
		}
		jwtName = strings.TrimSpace(jwtName)
		if _, ok := available[jwtName]; !ok {
			klog.Warningf("Ingress %s/%s references Entra JWT config %q which is not in Helm appgw.entraJWT and not present on Application Gateway; skipping",
				ingress.Namespace, ingress.Name, jwtName)
			continue
		}

		for listenerID := range c.getListenersFromIngress(ingress, cbCtx.EnvVariables) {
			cfg, ok := listenerConfigs[listenerID]
			if !ok || cfg.Protocol != n.ApplicationGatewayProtocolHTTPS {
				klog.Warningf("Ingress %s/%s Entra JWT config %q ignored for non-HTTPS listener %+v",
					ingress.Namespace, ingress.Name, jwtName, listenerID)
				continue
			}
			if conflicts[listenerID] {
				continue
			}
			if existing, ok := bindings[listenerID]; ok && existing != jwtName {
				klog.Errorf("Conflicting Entra JWT config names on shared listener %+v: %q vs %q; skipping JWT attachment", listenerID, existing, jwtName)
				delete(bindings, listenerID)
				conflicts[listenerID] = true
				continue
			}
			bindings[listenerID] = jwtName
		}
	}

	return bindings
}
