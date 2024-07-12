// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"encoding/json"
	"strings"

	"k8s.io/klog/v2"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// TargetBlacklist is a list of Targets, which AGIC is not allowed to apply configuration for.
type TargetBlacklist *[]Target

// TargetPath is a string type alias.
type TargetPath string

// Target uniquely identifies a subset of App Gateway configuration, which AGIC will manage or be prohibited from managing.
type Target struct {
	Hostname string     `json:"Hostname,omitempty"`
	Path     TargetPath `json:"Path,omitempty"`
}

// IsBlacklisted figures out whether a given Target objects in a list of blacklisted targets.
func (t Target) IsBlacklisted(blacklist TargetBlacklist) bool {
	jsonTarget, _ := json.Marshal(t)
	for _, blTarget := range *blacklist {

		// An empty blacklist hostname indicates that any hostname would be blacklisted.
		// If host names match - this target is in the blacklist.
		// AGIC is allowed to create and modify App Gwy config for blank host.
		hostIsBlacklisted := blTarget.Hostname == "" || strings.ToLower(t.Hostname) == strings.ToLower(blTarget.Hostname)

		pathIsBlacklisted := blTarget.Path == "" || blTarget.Path == "/*" || t.Path.lower() == blTarget.Path.lower() || blTarget.Path.contains(t.Path) // TODO(draychev): || t.Path.contains(blTarget.Path)

		// With this version we keep things as simple as possible: match host and exact path to determine
		// whether given target is in the blacklist. Ideally this would be URL Path set overlap operation,
		// which we deliberately leave for a later time.
		if hostIsBlacklisted && pathIsBlacklisted {
			klog.V(3).Infof("[brownfield] Target %s is blacklisted", jsonTarget)
			return true // Found it
		}
	}
	klog.V(3).Infof("[brownfield] Target %s is not blacklisted", jsonTarget)
	return false // Did not find it
}

// GetTargetBlacklist returns the list of Targets given a list ProhibitedTarget CRDs.
func GetTargetBlacklist(prohibitedTargets []*ptv1.AzureIngressProhibitedTarget) TargetBlacklist {
	// TODO(draychev): make this a method of ExistingResources and memoize it.
	var target []Target
	for _, prohibitedTarget := range prohibitedTargets {
		if len(prohibitedTarget.Spec.Paths) == 0 {
			target = append(target, Target{
				Hostname: prohibitedTarget.Spec.Hostname,
			})
		}
		for _, path := range prohibitedTarget.Spec.Paths {
			target = append(target, Target{
				Hostname: prohibitedTarget.Spec.Hostname,
				Path:     TargetPath(strings.ToLower(path)),
			})
		}
	}
	return &target
}

func (p TargetPath) lower() string {
	return strings.ToLower(string(p))
}

func (p TargetPath) contains(otherPath TargetPath) bool {
	if p == "" || p == "*" || p == "/*" {
		return true
	}

	// For strings that do not end with a * - do exact match
	if !strings.HasSuffix(p.lower(), "*") {
		return p.lower() == otherPath.lower()
	}

	// "/x/*" contains "/x"
	if strings.TrimRight(p.lower(), "/*") == strings.TrimRight(otherPath.lower(), "/*") {
		return true
	}

	if len(p) > len(otherPath) {
		return false
	}

	thisPathChunks := strings.Split(p.lower(), "/")
	otherPathChunks := strings.Split(otherPath.lower(), "/")
	for idx := range thisPathChunks {
		if thisPathChunks[idx] == "*" {
			return true
		}
		if thisPathChunks[idx] != otherPathChunks[idx] {
			return false
		}
	}
	return false
}
