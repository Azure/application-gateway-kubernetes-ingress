// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"encoding/json"
	"github.com/golang/glog"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// Target uniquely identifies a subset of App Gateway configuration, which AGIC will manage or be prohibited from managing.
type Target struct {
	Hostname string
	Port     int32
	Path     *string
}

// IsIn figures out whether a given Target objects in a list of Target objects.
func (t Target) IsIn(targetList *[]Target) bool {
	for _, otherTarget := range *targetList {
		hostIsSame := strings.ToLower(t.Hostname) == strings.ToLower(otherTarget.Hostname)

		// If one of the ports is not defined (0) - ignore port comparison
		portIsSame := t.Port == otherTarget.Port || t.Port == 0 || otherTarget.Port == 0

		// Set defaults to blank string, so we can compare strings even if nulls.
		pathA, pathB := "", ""
		if t.Path != nil {
			pathA = *t.Path
		}
		if otherTarget.Path != nil {
			pathB = *otherTarget.Path
		}

		if hostIsSame && portIsSame && pathA == pathB {
			// Found it
			return true
		}
	}

	// Did not find it
	return false
}

// prettyTarget is used for pretty-printing the Target struct for debugging purposes.
type prettyTarget struct {
	Hostname string `json:"Hostname"`
	Port     int32  `json:"Port"`
	Path     string `json:"Path,omitempty"`
}

// MarshalJSON converts the Target object to a JSON byte array.
func (t Target) MarshalJSON() ([]byte, error) {
	pt := prettyTarget{
		Hostname: t.Hostname,
		Port:     t.Port,
	}
	if t.Path != nil {
		pt.Path = *t.Path
	}
	return json.Marshal(pt)
}

// GetTargetBlacklist returns the list of Targets given a list ProhibitedTarget CRDs.
func GetTargetBlacklist(prohibitedTargets []*ptv1.AzureIngressProhibitedTarget) TargetBlacklist {
	var target []Target
	for _, prohibitedTarget := range prohibitedTargets {
		if len(prohibitedTarget.Spec.Paths) == 0 {
			target = append(target, Target{
				Hostname: prohibitedTarget.Spec.Hostname,
				Port:     prohibitedTarget.Spec.Port,
				Path:     nil,
			})
		}
		for _, path := range prohibitedTarget.Spec.Paths {
			target = append(target, Target{
				Hostname: prohibitedTarget.Spec.Hostname,
				Port:     prohibitedTarget.Spec.Port,
				Path:     to.StringPtr(NormalizePath(path)),
			})
		}
	}
	return &target
}

// GetTargetWhitelist returns the list of Targets given a list ManagedTarget CRDs.
func GetTargetWhitelist(managedTargets []*mtv1.AzureIngressManagedTarget) TargetWhitelist {
	var target []Target
	for _, managedTarget := range managedTargets {
		if len(managedTarget.Spec.Paths) == 0 {
			target = append(target, Target{
				Hostname: managedTarget.Spec.Hostname,
				Port:     managedTarget.Spec.Port,
				Path:     nil,
			})
		}
		for _, path := range managedTarget.Spec.Paths {
			target = append(target, Target{
				Hostname: managedTarget.Spec.Hostname,
				Port:     managedTarget.Spec.Port,
				Path:     to.StringPtr(NormalizePath(path)),
			})
		}
	}
	return &target
}

// NormalizePath re-formats the path string so that we can discovere semantically identical paths.
func NormalizePath(path string) string {
	trimmed, prevTrimmed := "", path
	cutset := "*/"
	for trimmed != prevTrimmed {
		prevTrimmed = trimmed
		trimmed = strings.TrimRight(path, cutset)
	}
	return trimmed
}

// shouldManage determines whether the target identified by the given host & path should be managed by AGIC.
func shouldManage(host string, path *string, blacklist TargetBlacklist, whitelist TargetWhitelist) bool {

	target := rulePathToTarget(host, path)

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

// rulePathToTarget constructs a Target struct based on the host and path provided
// TODO(draychev): Add port number to enable port-specific target management.
func rulePathToTarget(host string, path *string) *Target {
	target := Target{
		Hostname: host,
	}
	if path != nil {
		target.Path = path
	}
	return &target
}
