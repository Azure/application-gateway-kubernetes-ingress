// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayrewrite/v1beta1"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/klog/v2"
)

func (c *appGwConfigBuilder) Rewrites(cbCtx *ConfigBuilderContext) error {
	agrrs := c.getRewrites(cbCtx)
	c.appGw.RewriteRuleSets = &agrrs
	return nil
}

func (c appGwConfigBuilder) getRewrites(cbCtx *ConfigBuilderContext) []n.ApplicationGatewayRewriteRuleSet {

	if c.mem.rewrites != nil {
		return *c.mem.rewrites
	}

	var ret []n.ApplicationGatewayRewriteRuleSet

	for _, ingress := range cbCtx.IngressList {

		klog.V(1).Infof("Looking for rewrite-rule-set-crd annotation in %s/%s", ingress.Namespace, ingress.Name)

		rewriteRuleSetCRDName, err := annotations.RewriteRuleSetCRD(ingress)

		// if there is error fetching CRDName or the value is "", move onto the next ingress
		if err != nil || rewriteRuleSetCRDName == "" {
			continue
		}

		rewrite, err := c.k8sContext.GetRewrite(ingress.Namespace, rewriteRuleSetCRDName)

		if err != nil {
			klog.Error("Error occured while fetching rewrite CRD for", rewriteRuleSetCRDName)
			continue
		}

		ret = append(ret, c.convertRewrite(rewriteRuleSetCRDName, rewrite))
	}

	return ret
}

func (c appGwConfigBuilder) convertRewrite(rewriteRuleSetCRDName string, rewrite *v1beta1.AzureApplicationGatewayRewrite) n.ApplicationGatewayRewriteRuleSet {

	rewriteRules := []n.ApplicationGatewayRewriteRule{}

	for _, rr := range rewrite.Spec.RewriteRules {

		rewriteRule := n.ApplicationGatewayRewriteRule{
			Name:         to.StringPtr(rr.Name),
			RuleSequence: to.Int32Ptr(int32(rr.RuleSequence)),
			Conditions:   makeConditions(rr.Conditions),
			ActionSet:    makeActionSet(rr.Actions),
		}

		rewriteRules = append(rewriteRules, rewriteRule)
	}

	return n.ApplicationGatewayRewriteRuleSet{
		Name: to.StringPtr(rewriteRuleSetCRDName),
		ID:   to.StringPtr(c.appGwIdentifier.rewriteRuleSetID(rewriteRuleSetCRDName)),

		ApplicationGatewayRewriteRuleSetPropertiesFormat: &n.ApplicationGatewayRewriteRuleSetPropertiesFormat{
			RewriteRules: &rewriteRules,
		},
	}
}

// conversion functions
func makeConditions(Conditions []v1beta1.Condition) *[]n.ApplicationGatewayRewriteRuleCondition {

	ret := []n.ApplicationGatewayRewriteRuleCondition{}

	for _, Condition := range Conditions {
		ret = append(ret, n.ApplicationGatewayRewriteRuleCondition{
			IgnoreCase: to.BoolPtr(Condition.IgnoreCase),
			Negate:     to.BoolPtr(Condition.Negate),
			Variable:   to.StringPtr(Condition.Variable),
			Pattern:    to.StringPtr(Condition.Pattern),
		})
	}

	return &ret
}

func makeActionSet(Actions v1beta1.Actions) *n.ApplicationGatewayRewriteRuleActionSet {
	return &n.ApplicationGatewayRewriteRuleActionSet{
		RequestHeaderConfigurations:  makeHeaderConfigs(Actions.RequestHeaderConfigurations),
		ResponseHeaderConfigurations: makeHeaderConfigs(Actions.ResponseHeaderConfigurations),
		URLConfiguration:             makeURLConfig(Actions.UrlConfiguration),
	}
}

func makeHeaderConfigs(headerConfigs []v1beta1.HeaderConfiguration) *[]n.ApplicationGatewayHeaderConfiguration {

	ret := []n.ApplicationGatewayHeaderConfiguration{}

	for _, headerConfig := range headerConfigs {
		ret = append(ret, n.ApplicationGatewayHeaderConfiguration{
			HeaderName:  to.StringPtr(headerConfig.HeaderName),
			HeaderValue: to.StringPtr(headerConfig.HeaderValue),
		})
	}

	return &ret
}

func makeURLConfig(urlConfig v1beta1.UrlConfiguration) *n.ApplicationGatewayURLConfiguration {
	return &n.ApplicationGatewayURLConfiguration{
		ModifiedPath:        to.StringPtr(urlConfig.ModifiedPath),
		ModifiedQueryString: to.StringPtr(urlConfig.ModifiedQueryString),
		Reroute:             to.BoolPtr(urlConfig.Reroute),
	}
}
