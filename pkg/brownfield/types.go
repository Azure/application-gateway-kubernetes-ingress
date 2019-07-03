// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

// PoolContext is the basket of App Gateway configs necessary to determine what settings should be
// managed and what should be left as-is.
type PoolContext struct {
	Listeners    []*n.ApplicationGatewayHTTPListener
	RoutingRules []n.ApplicationGatewayRequestRoutingRule
	PathMaps     []n.ApplicationGatewayURLPathMap
}

type listenerName string
type pathmapName string
type backendPoolName string

type poolToTargets map[backendPoolName][]Target

type poolsByName map[backendPoolName]n.ApplicationGatewayBackendAddressPool

type TargetWhitelist *[]Target
type TargetBlacklist *[]Target
