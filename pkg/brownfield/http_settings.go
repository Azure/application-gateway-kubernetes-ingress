// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// TODO(draychev): COMBINE THIS WITH POOLS ETC
func GetSettingToTargetMapping(rr []n.ApplicationGatewayRequestRoutingRule, listeners ListenersByName, pathMap URLPathMapByName) NameToTarget {

	settingToTarget := make(map[string]Target)

	// TODO(draychev): COMBINE THIS WITH POOLS ETC
	for _, rule := range rr {
		listenerName := utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID)
		listener := listeners[listenerName]
		target := Target{
			Hostname: *listeners[listenerName].HostName,
			Port:     portFromListener(listener),
		}
		if rule.URLPathMap == nil {
			if rule.BackendHTTPSettings == nil {
				continue
			}
			settingToTarget[utils.GetLastChunkOfSlashed(*rule.BackendHTTPSettings.ID)] = target
		} else {
			pathMapName := utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID)
			for _, pathRule := range *pathMap[pathMapName].PathRules {
				if pathRule.BackendHTTPSettings == nil {
					continue
				}
				for _, path := range *pathRule.Paths {
					target.Path = &path
					settingToTarget[utils.GetLastChunkOfSlashed(*pathRule.BackendHTTPSettings.ID)] = target
				}
			}
		}
	}
	return settingToTarget
}

// TODO(draychev): COMBINE THIS WITH POOLS ETC
func GetManagedSettings(sett *[]n.ApplicationGatewayBackendHTTPSettings, routingRules []n.ApplicationGatewayRequestRoutingRule, listenersByName ListenersByName, pathMapsByName URLPathMapByName, managedTargets []*ptv1.AzureIngressManagedTarget, prohibitedTargets []*mtv1.AzureIngressProhibitedTarget) (*[]n.ApplicationGatewayBackendHTTPSettings, error) {
	blacklist := getProhibitedTargetList(prohibitedTargets)
	whitelist := getManagedTargetList(managedTargets)

	if len(*blacklist) == 0 && len(*whitelist) == 0 {
		return sett, nil
	}

	var managedSettings []n.ApplicationGatewayBackendHTTPSettings

	settingNameToTarget := GetSettingToTargetMapping(routingRules, listenersByName, pathMapsByName)

	// Blacklist takes priority
	if len(*blacklist) > 0 {
		// Apply blacklist
		for _, setting := range *sett {
			target := settingNameToTarget[*setting.Name]
			if target.IsIn(blacklist) {
				continue
			}
			managedSettings = append(managedSettings, setting)
		}
		return &managedSettings, nil
	}

	// Is it whitelisted
	for _, setting := range *sett {
		target := settingNameToTarget[*setting.Name]
		if target.IsIn(whitelist) {
			managedSettings = append(managedSettings, setting)
		}
	}

	return &managedSettings, nil
}
