// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"strings"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayrewrite/v1beta1"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/klog/v2"
)

// RewriteRuleSets builds the RewriteRuleSets part of the configBuilder
func (c *appGwConfigBuilder) RewriteRuleSets(cbCtx *ConfigBuilderContext) error {

	if c.appGw.RewriteRuleSets == nil {
		c.appGw.RewriteRuleSets = &[]n.ApplicationGatewayRewriteRuleSet{}
	}

	rewriteRuleSets := removeAGICGeneratedRewriteRuleSets(c.appGw.RewriteRuleSets)
	rewriteRuleSets = append(rewriteRuleSets, c.getAGICRewriteRuleSets(cbCtx)...)

	c.appGw.RewriteRuleSets = &rewriteRuleSets
	return nil
}

// removeAGICGeneratedRewriteRuleSets removes the rewrite rule sets that were generated by AGIC
func removeAGICGeneratedRewriteRuleSets(currentRewriteRuleSets *[]n.ApplicationGatewayRewriteRuleSet) []n.ApplicationGatewayRewriteRuleSet {

	if len(*currentRewriteRuleSets) == 0 {
		return *currentRewriteRuleSets
	}

	var appGwRewriteRuleSets []n.ApplicationGatewayRewriteRuleSet

	for _, rrs := range *currentRewriteRuleSets {
		if rewriteRuleSetName := *(rrs.Name); !(strings.HasPrefix(rewriteRuleSetName, "crd-")) {
			appGwRewriteRuleSets = append(appGwRewriteRuleSets, rrs)
		}
	}

	return appGwRewriteRuleSets
}

// getAGICRewriteRuleSets returns all the rewrite rule sets that are referenced in atleast one of the ingress manifests
func (c appGwConfigBuilder) getAGICRewriteRuleSets(cbCtx *ConfigBuilderContext) []n.ApplicationGatewayRewriteRuleSet {

	type rewriteCRDInfo struct {
		ingressNamespace string
		crdName          string
	}

	uniqueRewriteRuleSetCR := map[string]rewriteCRDInfo{}

	// insert all referenced rewrite rule sets into a map to avoid duplicates
	for _, ingress := range cbCtx.IngressList {

		klog.V(9).Infof("Looking for %s annotation in %s/%s", annotations.RewriteRuleSetCustomResourceKey, ingress.Namespace, ingress.Name)
		rewriteRuleSetCRName, err := annotations.RewriteRuleSetCustomResource(ingress)

		// if there is error fetching CR or if the value is "", move onto the next ingress
		if err != nil || rewriteRuleSetCRName == "" {
			continue
		}

		uniqueRewriteRuleSetCR[ingress.Namespace+"/"+rewriteRuleSetCRName] = rewriteCRDInfo{
			ingressNamespace: ingress.Namespace,
			crdName:          rewriteRuleSetCRName,
		}
	}

	var appGwRewriteRuleSet []n.ApplicationGatewayRewriteRuleSet

	// get details of all the unique rewrite rule sets referenced in various ingress manifest
	for _, rewriteRuleSetCRInfo := range uniqueRewriteRuleSetCR {

		rewrite, err := c.k8sContext.GetRewriteRuleSetCustomResource(rewriteRuleSetCRInfo.ingressNamespace, rewriteRuleSetCRInfo.crdName)

		if err != nil {
			klog.Errorf("Error occured while fetching rewrite rule set custom resource named %s.", rewriteRuleSetCRInfo.crdName)
			continue
		}

		appGwRewriteRuleSet = append(appGwRewriteRuleSet, c.makeRewrite(rewriteRuleSetCRInfo.ingressNamespace, rewriteRuleSetCRInfo.crdName, rewrite))
	}

	return appGwRewriteRuleSet
}

// c.makeRewrite converts *v1beta1.AzureApplicationGatewayRewrite to n.ApplicationGatewayRewriteRuleSet
func (c appGwConfigBuilder) makeRewrite(namespace string, rewriteRuleSetCRName string, rewrite *v1beta1.AzureApplicationGatewayRewrite) n.ApplicationGatewayRewriteRuleSet {

	// prefix AGIC built rewriteRuleSets by crd- to help differentiate from user created rewrite rule sets
	rewriteRuleSetCRName = fmt.Sprintf("crd-%s-%s", namespace, rewriteRuleSetCRName)

	appGwRewriteRules := []n.ApplicationGatewayRewriteRule{}

	for _, rr := range rewrite.Spec.RewriteRules {

		appGwNewRewriteRule := n.ApplicationGatewayRewriteRule{
			Name:         to.StringPtr(rr.Name),
			RuleSequence: to.Int32Ptr(int32(rr.RuleSequence)),
			Conditions:   makeConditions(rr.Conditions),
			ActionSet:    makeActionSet(rr.Actions),
		}

		appGwRewriteRules = append(appGwRewriteRules, appGwNewRewriteRule)
	}

	return n.ApplicationGatewayRewriteRuleSet{
		Name: to.StringPtr(rewriteRuleSetCRName),
		ID:   to.StringPtr(c.appGwIdentifier.rewriteRuleSetID(rewriteRuleSetCRName)),

		ApplicationGatewayRewriteRuleSetPropertiesFormat: &n.ApplicationGatewayRewriteRuleSetPropertiesFormat{
			RewriteRules: &appGwRewriteRules,
		},
	}
}

// makeConditions converts []v1beta.Condition to *[]n.ApplicationGatewayRewriteRuleCondition
func makeConditions(apiConditions []v1beta1.Condition) *[]n.ApplicationGatewayRewriteRuleCondition {

	appGwConditions := []n.ApplicationGatewayRewriteRuleCondition{}

	for _, c := range apiConditions {

		appGwConditions = append(appGwConditions, n.ApplicationGatewayRewriteRuleCondition{
			IgnoreCase: to.BoolPtr(c.IgnoreCase),
			Negate:     to.BoolPtr(c.Negate),
			Variable:   to.StringPtr(c.Variable),
			Pattern:    to.StringPtr(c.Pattern),
		})

	}

	return &appGwConditions
}

// makeActionSet converts v1beta1.Actions to *n.ApplicationGatewayRewriteRuleActionSet
func makeActionSet(apiActions v1beta1.Actions) *n.ApplicationGatewayRewriteRuleActionSet {

	return &n.ApplicationGatewayRewriteRuleActionSet{
		RequestHeaderConfigurations:  makeHeaderConfigs(apiActions.RequestHeaderConfigurations),
		ResponseHeaderConfigurations: makeHeaderConfigs(apiActions.ResponseHeaderConfigurations),
		URLConfiguration:             makeURLConfig(apiActions.UrlConfiguration),
	}

}

// makeHeaderConfigs converts []v1beta1.HeaderConfiguration to *[]n.ApplicationGatewayHeaderConfiguration
func makeHeaderConfigs(apiHeaderConfigs []v1beta1.HeaderConfiguration) *[]n.ApplicationGatewayHeaderConfiguration {

	appGwHeaderConfig := []n.ApplicationGatewayHeaderConfiguration{}

	for _, hc := range apiHeaderConfigs {

		appGwHeaderConfig = append(appGwHeaderConfig, n.ApplicationGatewayHeaderConfiguration{
			HeaderName:  to.StringPtr(hc.HeaderName),
			HeaderValue: to.StringPtr(hc.HeaderValue),
		})

	}

	return &appGwHeaderConfig
}

// makeURLConfig converts v1beta1.UrlConfiguration to *n.ApplicationGatewayURLConfiguration
func makeURLConfig(apiURLConfig *v1beta1.UrlConfiguration) *n.ApplicationGatewayURLConfiguration {
	if apiURLConfig == nil {
		return nil
	}

	return &n.ApplicationGatewayURLConfiguration{
		ModifiedPath:        to.StringPtr(apiURLConfig.ModifiedPath),
		ModifiedQueryString: to.StringPtr(apiURLConfig.ModifiedQueryString),
		Reroute:             to.BoolPtr(apiURLConfig.Reroute),
	}
}
