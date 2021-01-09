// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

type probeName string
type probesByName map[probeName]n.ApplicationGatewayProbe

// GetBlacklistedProbes filters the given list of health probes to the list Probes that AGIC is allowed to manage.
func (er ExistingResources) GetBlacklistedProbes() ([]n.ApplicationGatewayProbe, []n.ApplicationGatewayProbe) {
	blacklistedProbesSet := er.getBlacklistedProbesSet()
	var nonBlacklistedProbes []n.ApplicationGatewayProbe
	var blacklistedProbes []n.ApplicationGatewayProbe
	for _, probe := range er.Probes {
		if _, isBlacklisted := blacklistedProbesSet[probeName(*probe.Name)]; isBlacklisted {
			klog.V(5).Infof("Probe %s is blacklisted", *probe.Name)
			blacklistedProbes = append(blacklistedProbes, probe)
			continue
		}
		klog.V(5).Infof("Probe %s is not blacklisted", *probe.Name)
		nonBlacklistedProbes = append(nonBlacklistedProbes, probe)
	}
	return blacklistedProbes, nonBlacklistedProbes
}

// GetNotWhitelistedProbes filters the given list of health probes to the list Probes that AGIC is allowed to manage.
func (er ExistingResources) GetNotWhitelistedProbes() ([]n.ApplicationGatewayProbe, []n.ApplicationGatewayProbe) {
	whitelistedProbesSet := er.getWhitelistedProbesSet()
	var nonWhitelistedProbes []n.ApplicationGatewayProbe
	var whitelistedProbes []n.ApplicationGatewayProbe
	for _, probe := range er.Probes {
		if _, isWhitelisted := whitelistedProbesSet[probeName(*probe.Name)]; isWhitelisted {
			klog.V(5).Infof("Probe %s is whitelisted", *probe.Name)
			whitelistedProbes = append(whitelistedProbes, probe)
			continue
		}
		klog.V(5).Infof("Probe %s is not whitelisted", *probe.Name)
		nonWhitelistedProbes = append(nonWhitelistedProbes, probe)
	}
	return nonWhitelistedProbes, whitelistedProbes
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
func LogProbes(logger Logger, existingBlacklisted []n.ApplicationGatewayProbe, existingNonBlacklisted []n.ApplicationGatewayProbe, managedProbes []n.ApplicationGatewayProbe) {
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

	logger.Info("[brownfield] Probes AGIC created: ", getProbeNames(managedProbes))
	logger.Info("[brownfield] Existing Blacklisted Probes AGIC will retain: ", getProbeNames(existingBlacklisted))
	logger.Info("[brownfield] Existing Probes AGIC will remove: ", getProbeNames(garbage))
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

func (er ExistingResources) getBlacklistedProbesSet() map[probeName]interface{} {
	blacklistedHTTPSettings, _ := er.GetBlacklistedHTTPSettings()
	blacklistedProbesSet := make(map[probeName]interface{})
	for _, setting := range blacklistedHTTPSettings {
		if setting.Probe != nil && setting.Probe.ID != nil {
			probeName := probeName(utils.GetLastChunkOfSlashed(*setting.Probe.ID))
			blacklistedProbesSet[probeName] = nil
		}
	}
	return blacklistedProbesSet
}

func (er ExistingResources) getWhitelistedProbesSet() map[probeName]interface{} {
	_, whitelistedHTTPSettings := er.GetNotWhitelistedHTTPSettings()
	whitelistedProbesSet := make(map[probeName]interface{})
	for _, setting := range whitelistedHTTPSettings {
		if setting.Probe != nil && setting.Probe.ID != nil {
			probeName := probeName(utils.GetLastChunkOfSlashed(*setting.Probe.ID))
			whitelistedProbesSet[probeName] = nil
		}
	}
	return whitelistedProbesSet
}
