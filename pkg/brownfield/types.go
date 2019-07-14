// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

type listenerName string

type pathmapName string
type poolToTargets map[backendPoolName][]Target

// ExistingResources is used in brownfield deployments and
// holds a copy of the existing App Gateway config, based
// on which AGIC will determine what should be retained and
// what config should be discarded or overwritten.
type ExistingResources struct {
	BackendPools       []n.ApplicationGatewayBackendAddressPool
	Certificates       []n.ApplicationGatewaySslCertificate
	RoutingRules       []n.ApplicationGatewayRequestRoutingRule
	Listeners          []n.ApplicationGatewayHTTPListener
	URLPathMaps        []n.ApplicationGatewayURLPathMap
	HTTPSettings       []n.ApplicationGatewayBackendHTTPSettings
	Ports              []n.ApplicationGatewayFrontendPort
	Probes             []n.ApplicationGatewayProbe
	ProhibitedTargets  []*v1.AzureIngressProhibitedTarget
	DefaultBackendPool *n.ApplicationGatewayBackendAddressPool

	// Cache helper structs
	listenersByName   map[listenerName]n.ApplicationGatewayHTTPListener
	urlPathMapsByName pathMapsByName
}

// NewExistingResources creates a new ExistingResources struct.
func NewExistingResources(appGw n.ApplicationGateway, prohibitedTargets []*v1.AzureIngressProhibitedTarget, defaultPool *n.ApplicationGatewayBackendAddressPool) ExistingResources {
	var allExistingSettings []n.ApplicationGatewayBackendHTTPSettings
	if appGw.BackendHTTPSettingsCollection != nil {
		allExistingSettings = *appGw.BackendHTTPSettingsCollection
	}

	var allExistingRequestRoutingRules []n.ApplicationGatewayRequestRoutingRule
	if appGw.RequestRoutingRules != nil {
		allExistingRequestRoutingRules = *appGw.RequestRoutingRules
	}

	var allExistingListeners []n.ApplicationGatewayHTTPListener
	if appGw.HTTPListeners != nil {
		allExistingListeners = *appGw.HTTPListeners
	}

	var allExistingURLPathMap []n.ApplicationGatewayURLPathMap
	if appGw.URLPathMaps != nil {
		allExistingURLPathMap = *appGw.URLPathMaps
	}

	var allExistingPorts []n.ApplicationGatewayFrontendPort
	if appGw.FrontendPorts != nil {
		allExistingPorts = *appGw.FrontendPorts
	}

	var allExistingCertificates []n.ApplicationGatewaySslCertificate
	if appGw.SslCertificates != nil {
		allExistingCertificates = *appGw.SslCertificates
	}

	var allExistingHealthProbes []n.ApplicationGatewayProbe
	if appGw.Probes != nil {
		allExistingHealthProbes = *appGw.Probes
	}

	var allExistingBackendPools []n.ApplicationGatewayBackendAddressPool
	if appGw.BackendAddressPools != nil {
		allExistingBackendPools = *appGw.BackendAddressPools
	}

	return ExistingResources{
		BackendPools:       allExistingBackendPools,
		Certificates:       allExistingCertificates,
		RoutingRules:       allExistingRequestRoutingRules,
		Listeners:          allExistingListeners,
		URLPathMaps:        allExistingURLPathMap,
		HTTPSettings:       allExistingSettings,
		Ports:              allExistingPorts,
		Probes:             allExistingHealthProbes,
		ProhibitedTargets:  prohibitedTargets,
		DefaultBackendPool: defaultPool,
	}
}
