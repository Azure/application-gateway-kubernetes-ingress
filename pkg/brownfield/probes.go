// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

func GetManagedProbes(probes []n.ApplicationGatewayProbe, managedTargets []*mtv1.AzureIngressManagedTarget, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget) []n.ApplicationGatewayProbe {
	blacklist := getProhibitedTargetList(prohibitedTargets)
	whitelist := getManagedTargetList(managedTargets)

	if len(*blacklist) == 0 && len(*whitelist) == 0 {
		return probes
	}

	var managedProbes []n.ApplicationGatewayProbe

	// Blacklist takes priority
	if len(*blacklist) > 0 {
		// Apply blacklist
		for _, probe := range probes {
			if inProbeList(&probe, blacklist) {
				continue
			}
			managedProbes = append(managedProbes, probe)
		}
		return managedProbes
	}

	// Is it Whitelisted
	for _, probe := range probes {
		if inProbeList(&probe, whitelist) {
			managedProbes = append(managedProbes, probe)
		}
	}

	for _, probe := range probes {
		if inProbeList(&probe, blacklist) {
			managedProbes = append(managedProbes, probe)
		}
	}
	return managedProbes
}

func inProbeList(probe *n.ApplicationGatewayProbe, targetList *[]Target) bool {
	for _, t := range *targetList {
		if t.Hostname == *probe.Host {
			if t.Path == nil {
				// Host matches; No paths - found it
				return true
			} else if normalizePath(*t.Path) == normalizePath(*probe.Path) {
				// Matches a path - found it
				return true
			}
		}
	}

	// Did not find it
	return false
}

func MergeProbes(probesBuckets ...[]n.ApplicationGatewayProbe) []n.ApplicationGatewayProbe {
	uniqProbes := make(map[string]n.ApplicationGatewayProbe)
	for _, bucket := range probesBuckets {
		for _, p := range bucket {
			uniqProbes[*p.Name] = p
		}
	}
	var merged []n.ApplicationGatewayProbe
	for _, probe := range uniqProbes {
		merged = append(merged, probe)
	}
	return merged
}

func PruneManagedProbes(probes []n.ApplicationGatewayProbe, managedTargets []*mtv1.AzureIngressManagedTarget, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget) []n.ApplicationGatewayProbe {
	manageable := GetManagedProbes(probes, managedTargets, prohibitedTargets)
	if manageable == nil {
		return probes
	}
	indexed := make(map[string]n.ApplicationGatewayProbe)
	for _, probe := range manageable {
		indexed[*probe.Name] = probe
	}
	var unmanagedProbes []n.ApplicationGatewayProbe
	for _, probe := range probes {
		if _, isManaged := indexed[*probe.Name]; !isManaged {
			unmanagedProbes = append(unmanagedProbes, probe)
		}
	}
	return unmanagedProbes
}
