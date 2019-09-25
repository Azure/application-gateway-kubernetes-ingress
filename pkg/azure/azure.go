// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
)

// SubscriptionID is the subscription of the resource in the resourceID
type SubscriptionID string

// ResourceGroup is the resource group in which resource is deployed in the resourceID
type ResourceGroup string

// ResourceName is the resource name in the resourceID
type ResourceName string

// ParseResourceID gets subscriptionId, resource group, resource name from resourceID
func ParseResourceID(ID string) (SubscriptionID, ResourceGroup, ResourceName) {
	split := strings.Split(ID, "/")
	if len(split) < 9 {
		glog.Errorf("resourceID %s is invalid. There should be atleast 9 segments in resourceID", ID)
		return "", "", ""
	}

	return SubscriptionID(split[2]), ResourceGroup(split[4]), ResourceName(split[8])
}

// ConvertToClusterResourceGroup converts infra resource group to aks cluster ID
func ConvertToClusterResourceGroup(subscriptionID SubscriptionID, resourceGroup ResourceGroup, err error) (string, error) {
	if err != nil {
		return "", err
	}

	split := strings.Split(string(resourceGroup), "_")
	if len(split) != 4 || strings.ToUpper(split[0]) != "MC" {
		logLine := fmt.Sprintf("infrastructure resource group name: %s is expected to be of format MC_ResourceGroup_ResourceName_Location", string(resourceGroup))
		glog.Error(logLine)
		return "", ErrMissingResourceGroup
	}

	return fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ContainerService/managedClusters/%s", subscriptionID, split[1], split[2]), nil
}
