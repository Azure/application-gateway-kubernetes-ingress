package brownfield

import (
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
)

// PruneIngress mutates the ingress object to prune prohibited targets and leaves ones AGIC will manage.
func PruneIngress(ing *v1beta1.Ingress, blacklist TargetBlacklist, whitelist TargetWhitelist) {
	var rules []v1beta1.IngressRule

	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}

		target := Target{
			Hostname: rule.Host,
			Port:     443, // TODO(delyan) !!!!!
			Path:     nil,
		}

		if rule.HTTP.Paths == nil {
			if shouldKeep(target, blacklist, whitelist) {
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
			target.Path = &path.Path
			if shouldKeep(target, blacklist, whitelist) {
				newRule.HTTP.Paths = append(newRule.HTTP.Paths, path)
			}
		}
		if len(newRule.HTTP.Paths) > 0 {
			rules = append(rules, newRule)
		}
	}

	ing.Spec.Rules = rules
}

func shouldKeep(target Target, blacklist TargetBlacklist, whitelist TargetWhitelist) bool {
	// Apply Blacklist first to remove explicitly forbidden targets.
	if blacklist != nil && len(*blacklist) > 0 {
		targetJSON, _ := target.MarshalJSON()
		if target.IsIn(blacklist) {
			glog.V(5).Infof("Target is in blacklist. Ignore: %s", string(targetJSON))
			return false
		}
		glog.V(5).Infof("Target is not in blacklist. Keep: %s", string(targetJSON))
		return true
	}

	if whitelist != nil && len(*whitelist) > 0 {
		targetJSON, _ := target.MarshalJSON()
		if target.IsIn(whitelist) {
			glog.V(5).Infof("Target is in the whitelist. Keep: %s", string(targetJSON))
			return true
		}
		glog.V(5).Infof("Target is not in the whitelist. Ignore: %s", string(targetJSON))
		return false
	}

	//There's neither blacklist nor whitelist - keep it
	return true
}
