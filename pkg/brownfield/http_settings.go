// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

type settingName string
type settingsByName map[settingName]n.ApplicationGatewayBackendHTTPSettings

// GetBlacklistedHTTPSettings filters the given list of routing pathMaps to the list pathMaps that AGIC is allowed to manage.
// HTTP Setting is blacklisted when it is associated with a Routing Rule that is blacklisted.
func (er ExistingResources) GetBlacklistedHTTPSettings() ([]n.ApplicationGatewayBackendHTTPSettings, []n.ApplicationGatewayBackendHTTPSettings) {
	blacklistedSettingsSet := er.getBlacklistedSettingsSet()
	var blacklisted []n.ApplicationGatewayBackendHTTPSettings
	var nonBlacklisted []n.ApplicationGatewayBackendHTTPSettings
	for _, setting := range er.HTTPSettings {
		if _, isBlacklisted := blacklistedSettingsSet[settingName(*setting.Name)]; isBlacklisted {
			blacklisted = append(blacklisted, setting)
			klog.V(5).Infof("HTTP Setting %s is blacklisted", *setting.Name)
			continue
		}
		klog.V(5).Infof("HTTP Setting %s is NOT blacklisted", *setting.Name)
		nonBlacklisted = append(nonBlacklisted, setting)
	}
	return blacklisted, nonBlacklisted
}

// MergeHTTPSettings merges list of lists of HTTP Settings into a single list, maintaining uniqueness.
func MergeHTTPSettings(settingBuckets ...[]n.ApplicationGatewayBackendHTTPSettings) []n.ApplicationGatewayBackendHTTPSettings {
	uniq := make(map[string]n.ApplicationGatewayBackendHTTPSettings)
	for _, bucket := range settingBuckets {
		for _, setting := range bucket {
			uniq[*setting.Name] = setting
		}
	}
	var merged []n.ApplicationGatewayBackendHTTPSettings
	for _, setting := range uniq {
		merged = append(merged, setting)
	}
	return merged
}

// LogHTTPSettings emits a few log lines detailing what settings are created, blacklisted, and removed from ARM.
func LogHTTPSettings(logger Logger, existingBlacklisted []n.ApplicationGatewayBackendHTTPSettings, existingNonBlacklisted []n.ApplicationGatewayBackendHTTPSettings, managedSettings []n.ApplicationGatewayBackendHTTPSettings) {
	var garbage []n.ApplicationGatewayBackendHTTPSettings

	blacklistedSet := indexSettingsByName(existingBlacklisted)
	managedSet := indexSettingsByName(managedSettings)

	for settingName, setting := range indexSettingsByName(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[settingName]
		_, existsInNewSettings := managedSet[settingName]
		if !existsInBlacklist && !existsInNewSettings {
			garbage = append(garbage, setting)
		}
	}

	logger.Info("[brownfield] HTTP Settings AGIC created: ", getSettingNames(managedSettings))
	logger.Info("[brownfield] Existing Blacklisted HTTP Settings AGIC will retain: ", getSettingNames(existingBlacklisted))
	logger.Info("[brownfield] Existing HTTP Settings AGIC will remove: ", getSettingNames(garbage))
}

func indexSettingsByName(settings []n.ApplicationGatewayBackendHTTPSettings) settingsByName {
	settingsByName := make(settingsByName)
	for _, setting := range settings {
		settingsByName[settingName(*setting.Name)] = setting
	}
	return settingsByName
}

func getSettingNames(settings []n.ApplicationGatewayBackendHTTPSettings) string {
	var names []string
	for _, setting := range settings {
		names = append(names, *setting.Name)
	}
	return strings.Join(names, ", ")
}

func (er ExistingResources) getBlacklistedSettingsSet() map[settingName]interface{} {
	blacklistedRoutingRules, _ := er.GetBlacklistedRoutingRules()
	blacklistedSettingsSet := make(map[settingName]interface{})
	for _, rule := range blacklistedRoutingRules {
		if rule.BackendHTTPSettings != nil && rule.BackendHTTPSettings.ID != nil {
			settingName := settingName(utils.GetLastChunkOfSlashed(*rule.BackendHTTPSettings.ID))
			blacklistedSettingsSet[settingName] = nil
		}
	}

	blacklistedPathMaps, _ := er.GetBlacklistedPathMaps()
	for _, pathMap := range blacklistedPathMaps {
		if pathMap.DefaultBackendAddressPool != nil && pathMap.DefaultBackendAddressPool.ID != nil {
			settingName := settingName(utils.GetLastChunkOfSlashed(*pathMap.DefaultBackendHTTPSettings.ID))
			blacklistedSettingsSet[settingName] = nil
		}
		if pathMap.PathRules == nil {
			klog.Errorf("PathMap %s does not have PathRules", *pathMap.Name)
			continue
		}
		for _, rule := range *pathMap.PathRules {
			if rule.BackendAddressPool != nil && rule.BackendAddressPool.ID != nil {
				settingName := settingName(utils.GetLastChunkOfSlashed(*rule.BackendHTTPSettings.ID))
				blacklistedSettingsSet[settingName] = nil
			}
		}
	}

	return blacklistedSettingsSet
}
