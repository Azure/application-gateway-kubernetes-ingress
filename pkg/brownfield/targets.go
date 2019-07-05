// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"encoding/json"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// Target uniquely identifies a subset of App Gateway configuration, which AGIC will manage or be prohibited from managing.
type Target struct {
	Hostname string
	Port     int32
	Path     *string
}

// IsBlacklisted figures out whether a given Target objects in a list of blacklisted targets.
func (t Target) IsBlacklisted(blacklist *[]Target) bool {
	for _, blTarget := range *blacklist {
		hostIsSame := strings.ToLower(t.Hostname) == strings.ToLower(blTarget.Hostname)

		// If one of the ports is not defined (0) - ignore port comparison
		portIsSame := t.Port == blTarget.Port || t.Port == 0 || blTarget.Port == 0

		// Set defaults to blank string, so we can compare strings even if nulls.
		targetPath, blacklistPath := "", ""
		if t.Path != nil {
			targetPath = *t.Path
		}
		if blTarget.Path != nil {
			blacklistPath = *blTarget.Path
		}

		if hostIsSame && portIsSame && pathsOverlap(targetPath, blacklistPath) {
			// Found it
			return true
		}
	}

	// Did not find it
	return false
}

// pathsOverlap determines whether 2 paths have any overlap.
// Example:  /a/b  and /a/b/c overlap;  /a/b and /a/x don't overlap.
func pathsOverlap(needle string, haystack string) bool {
	needle = NormalizePath(needle)
	haystack = NormalizePath(haystack)

	if needle == haystack {
		return true
	}

	needleChunks := strings.Split(needle, "/")
	haystackChunks := strings.Split(haystack, "/")

	for idx := 0; idx <= int(max(len(needleChunks), len(haystackChunks))); idx++ {
		if len(needleChunks) == idx || len(haystackChunks) == idx {
			return true
		}

		if needleChunks[idx] != haystackChunks[idx] {
			return false
		}
	}
	return true
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
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

// NormalizePath re-formats the path string so that we can discover semantically identical paths.
func NormalizePath(path string) string {
	trimmed, prevTrimmed := "", path
	cutset := "*/"
	for trimmed != prevTrimmed {
		prevTrimmed = trimmed
		trimmed = strings.TrimRight(path, cutset)
	}
	return strings.ToLower(trimmed)
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
