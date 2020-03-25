// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package environment

import "errors"

var (
	// ErrorMissingApplicationGatewayNameOrApplicationGatewayID is an error.
	ErrorMissingApplicationGatewayNameOrApplicationGatewayID = errors.New("Missing required Environment variables: " +
		"Provide atleast provide APPGW_NAME (helm var name: .appgw.name) or APPGW_RESOURCE_ID (helm var name: .appgw.applicationGatewayID). " +
		"If providing APPGW_NAME, You can also provided APPGW_SUBSCRIPTION_ID (helm var name: .appgw.subscriptionId) and APPGW_RESOURCE_GROUP (helm var name: .appgw.resourceGroup) (ENVT001)")

	// ErrorMissingApplicationgatewayName is an error.
	ErrorMissingApplicationgatewayName = errors.New("Missing required Environment variables: AGIC requires APPGW_NAME (helm var name: appgw.name) to deploy Application Gateway (ENVT002)")

	// ErrorNotAllowedApplicationgatewayID is an error.
	ErrorNotAllowedApplicationgatewayID = errors.New("Please provide provide APPGW_NAME (helm var name: .appgw.name) instead of APPGW_RESOURCE_ID (helm var name: .appgw.applicationGatewayID). " +
		"You can also provided APPGW_SUBSCRIPTION_ID and APPGW_RESOURCE_GROUP (ENVT003)")

	// ErrorMissingSubnetInfo is an error.
	ErrorMissingSubnetInfo = errors.New("Missing required Environment variables: " +
		"AGIC requires APPGW_SUBNET_PREFIX (helm var name: appgw.subnetPrefix) or APPGW_SUBNET_ID (helm var name: appgw.subnetID) of an existing subnet. " +
		"If subnetPrefix is specified, AGIC will look up a subnet with matching address prefix in the AKS cluster vnet. " +
		"If a subnet is not found, then a new subnet will be created. This will be used to deploy the Application Gateway (ENVT004)")

	// ErrorInvalidReconcilePeriod is an error.
	ErrorInvalidReconcilePeriod = errors.New("Please make sure that periodic reconcile is an integer. Range: (30 - 300) (ENVT005)")
)
