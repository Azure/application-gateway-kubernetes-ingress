// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// Logger is an abstraction over a logging facility.
type Logger interface {
	// Info is the a function allowing us to log messages.
	Info(args ...interface{})
}

type listenerName string

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
	Redirects          []n.ApplicationGatewayRedirectConfiguration
	ProhibitedTargets  []*ptv1.AzureIngressProhibitedTarget
	DefaultBackendPool *n.ApplicationGatewayBackendAddressPool

	// Cache helper structs
	listenersByName   map[listenerName]n.ApplicationGatewayHTTPListener
	urlPathMapsByName pathMapsByName
}

// NewExistingResources creates a new ExistingResources struct.
func NewExistingResources(appGw n.ApplicationGateway, prohibitedTargets []*ptv1.AzureIngressProhibitedTarget, defaultPool *n.ApplicationGatewayBackendAddressPool) ExistingResources {
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

	var allExistingRedirects []n.ApplicationGatewayRedirectConfiguration
	if appGw.RedirectConfigurations != nil {
		allExistingRedirects = *appGw.RedirectConfigurations
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
		Redirects:          allExistingRedirects,
		ProhibitedTargets:  prohibitedTargets,
		DefaultBackendPool: defaultPool,
	}
}

func (er ExistingResources) getProhibitedHostNames() map[string]interface{} {
	prohibitedHostNames := make(map[string]interface{})
	for _, pt := range er.ProhibitedTargets {
		if len(pt.Spec.Hostname) == 0 {
			continue
		}
		prohibitedHostNames[pt.Spec.Hostname] = nil
	}
	return prohibitedHostNames
}
