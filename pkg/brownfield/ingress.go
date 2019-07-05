// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// PruneIngressRules transforms the given ingress struct to remove targets, which AGIC should not create configuration for.
func PruneIngressRules(ing *v1beta1.Ingress, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget) []v1beta1.IngressRule {

	if ing.Spec.Rules == nil || len(ing.Spec.Rules) == 0 {
		return ing.Spec.Rules
	}

	blacklist := GetTargetBlacklist(prohibitedTargets)

	if blacklist == nil || len(*blacklist) == 0 {
		return ing.Spec.Rules
	}

	var rules []v1beta1.IngressRule

	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		if rule.HTTP.Paths == nil {
			if canManage(rule.Host, nil, blacklist) {
				rules = append(rules, rule)
			}
			continue // to next rule
		}

		newRule := v1beta1.IngressRule{
			Host: rule.Host,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{},
				},
			},
		}
		for _, path := range rule.HTTP.Paths {
			if canManage(rule.Host, &path.Path, blacklist) {
				newRule.HTTP.Paths = append(newRule.HTTP.Paths, path)
			}
		}
		if len(newRule.HTTP.Paths) > 0 {
			rules = append(rules, newRule)
		}
	}

	return rules
}

// canManage determines whether the target identified by the given host & path should be managed by AGIC.
func canManage(host string, path *string, blacklist TargetBlacklist) bool {
	target := Target{
		Hostname: host,
	}
	if path != nil {
		target.Path = path
	}

	if blacklist == nil || len(*blacklist) == 0 {
		return true
	}
	targetJSON, _ := target.MarshalJSON()
	if target.IsBlacklisted(blacklist) {
		glog.V(5).Infof("Target is in blacklist. Ignore: %s", string(targetJSON))
		return false
	}
	glog.V(5).Infof("Target is not in blacklist. Keep: %s", string(targetJSON))
	return true
}
