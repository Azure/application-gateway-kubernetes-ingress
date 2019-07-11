// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// GetBlacklistedProbes filters the given list of health probes to the list Probes that AGIC is allowed to manage.
func GetBlacklistedProbes(probes []n.ApplicationGatewayProbe, prohibited []*ptv1.AzureIngressProhibitedTarget) ([]n.ApplicationGatewayProbe, []n.ApplicationGatewayProbe) {
	blacklist := GetTargetBlacklist(prohibited)
	if len(*blacklist) == 0 {
		return nil, probes
	}
	var nonBlacklistedProbes []n.ApplicationGatewayProbe
	var blacklistedProbes []n.ApplicationGatewayProbe
	for _, probe := range probes {
		if inProbeBlacklist(&probe, blacklist) {
			logProbe(5, probe, "in blacklist")
			blacklistedProbes = append(blacklistedProbes, probe)
			continue
		}
		logProbe(5, probe, "not blacklisted")
		nonBlacklistedProbes = append(nonBlacklistedProbes, probe)
	}
	return blacklistedProbes, nonBlacklistedProbes
}

// MergeProbes merges list of lists of health probes into a single list, maintaining uniqueness.
func MergeProbes(probesBuckets ...[]n.ApplicationGatewayProbe) []n.ApplicationGatewayProbe {
	uniqProbes := make(probesByName)
	for _, bucket := range probesBuckets {
		for _, probe := range bucket {
			uniqProbes[probeName(*probe.Name)] = probe
		}
	}
	var managedProbes []n.ApplicationGatewayProbe
	for _, probe := range uniqProbes {
		managedProbes = append(managedProbes, probe)
	}
	return managedProbes
}

// LogProbes emits a few log lines detailing what probes are created, blacklisted, and removed from ARM.
func LogProbes(existingBlacklisted []n.ApplicationGatewayProbe, existingNonBlacklisted []n.ApplicationGatewayProbe, managedProbes []n.ApplicationGatewayProbe) {
	var garbage []n.ApplicationGatewayProbe

	blacklistedSet := indexProbesByName(existingBlacklisted)
	managedSet := indexProbesByName(managedProbes)

	for probeName, probe := range indexProbesByName(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[probeName]
		_, existsInNewProbes := managedSet[probeName]
		if !existsInBlacklist && !existsInNewProbes {
			garbage = append(garbage, probe)
		}
	}

	glog.V(3).Info("[brownfield] Probes AGIC created: ", getProbeNames(managedProbes))
	glog.V(3).Info("[brownfield] Existing Blacklisted Probes AGIC will retain: ", getProbeNames(existingBlacklisted))
	glog.V(3).Info("[brownfield] Existing Probes AGIC will remove: ", getProbeNames(garbage))
}

func logProbe(verbosity glog.Level, probe n.ApplicationGatewayProbe, message string) {
	t, _ := probe.MarshalJSON()
	glog.V(verbosity).Infof("Probe %s is "+message+": %s", *probe.Name, t)
}

func inProbeBlacklist(probe *n.ApplicationGatewayProbe, blacklist TargetBlacklist) bool {
	for _, target := range *blacklist {
		if target.Hostname == "" || target.Hostname == *probe.Host {
			if target.Path == "" {
				// Host matches; No paths - it is blacklisted
				return true
			} else if strings.HasPrefix(*probe.Path, strings.TrimRight(target.Path, "/*")) {
				// Matches a path or sub-path - it is blacklisted
				// If the target is: /abc -- will match probes for "/abc", as well as "/abc/healthz"
				return true
			}
		}
	}

	// Did not find it - is not blacklisted
	return false
}

func indexProbesByName(probes []n.ApplicationGatewayProbe) probesByName {
	probesByName := make(probesByName)
	for _, probe := range probes {
		probesByName[probeName(*probe.Name)] = probe
	}
	return probesByName
}

func getProbeNames(Probe []n.ApplicationGatewayProbe) string {
	var names []string
	for _, p := range Probe {
		names = append(names, *p.Name)
	}
	return strings.Join(names, ", ")
}
