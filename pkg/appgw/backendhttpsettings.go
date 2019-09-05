// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/errors"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

const (
	// DefaultConnDrainTimeoutInSec provides default value for ConnectionDrainTimeout
	DefaultConnDrainTimeoutInSec = 30
)

func (c *appGwConfigBuilder) BackendHTTPSettingsCollection(cbCtx *ConfigBuilderContext) error {
	agicHTTPSettings, _, _, err := c.getBackendsAndSettingsMap(cbCtx)

	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		rCtx := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)
		allExistingSettings := rCtx.HTTPSettings

		// PathMaps we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
		existingBlacklisted, existingNonBlacklisted := rCtx.GetBlacklistedHTTPSettings()

		brownfield.LogHTTPSettings(glog.V(3), existingBlacklisted, existingNonBlacklisted, agicHTTPSettings)

		// MergePathMaps would produce unique list of routing rules based on Name. Routing rules, which have the same name
		// as a managed rule would be overwritten.
		agicHTTPSettings = brownfield.MergeHTTPSettings(allExistingSettings, agicHTTPSettings)
	}
	if cbCtx.EnvVariables.EnableIstioIntegration {
		istioHTTPSettings, _, _, _ := c.getIstioDestinationsAndSettingsMap(cbCtx)
		if istioHTTPSettings != nil {
			sort.Sort(sorter.BySettingsName(istioHTTPSettings))
		}
		agicHTTPSettings = append(agicHTTPSettings, istioHTTPSettings...)
	}

	if agicHTTPSettings != nil {
		sort.Sort(sorter.BySettingsName(agicHTTPSettings))
	}

	c.appGw.BackendHTTPSettingsCollection = &agicHTTPSettings
	return err
}

func (c *appGwConfigBuilder) newBackendIdsFiltered(cbCtx *ConfigBuilderContext) map[backendIdentifier]interface{} {
	if c.mem.backendIDs != nil {
		return *c.mem.backendIDs
	}

	backendIDs := make(map[backendIdentifier]interface{})
	for _, ingress := range cbCtx.IngressList {
		if ingress.Spec.Backend != nil {
			backendID := generateBackendID(ingress, nil, nil, ingress.Spec.Backend)
			glog.V(3).Info("Found default backend:", backendID.serviceKey())
			backendIDs[backendID] = nil
		}
		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			if rule.HTTP == nil {
				// skip no http rule
				glog.V(5).Infof("[%s] Skip rule #%d for host '%s' - it has no HTTP rules.", ingress.Namespace, ruleIdx+1, rule.Host)
				continue
			}
			for pathIdx := range rule.HTTP.Paths {
				path := &rule.HTTP.Paths[pathIdx]
				backendID := generateBackendID(ingress, rule, path, &path.Backend)
				glog.V(5).Info("Found backend:", backendID.serviceKey())
				backendIDs[backendID] = nil
			}
		}
	}

	finalBackendIDs := make(map[backendIdentifier]interface{})
	serviceSet := newServiceSet(&cbCtx.ServiceList)
	// Filter out backends, where Ingresses reference non-existent Services
	for be := range backendIDs {
		if _, exists := serviceSet[be.serviceKey()]; !exists {
			glog.Errorf("Ingress %s/%s references non existent Service %s. Please correct the Service section of your Kubernetes YAML", be.Ingress.Namespace, be.Ingress.Name, be.serviceKey())
			// TODO(draychev): Enable this filter when we are certain this won't break anything!
			// continue
		}
		finalBackendIDs[be] = nil
	}

	c.mem.backendIDs = &finalBackendIDs
	return finalBackendIDs
}

func newServiceSet(services *[]*v1.Service) map[string]*v1.Service {
	servicesSet := make(map[string]*v1.Service)
	for _, service := range *services {
		serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
		servicesSet[serviceKey] = service
	}
	return servicesSet
}

func (c *appGwConfigBuilder) getBackendsAndSettingsMap(cbCtx *ConfigBuilderContext) ([]n.ApplicationGatewayBackendHTTPSettings, map[backendIdentifier]*n.ApplicationGatewayBackendHTTPSettings, map[backendIdentifier]serviceBackendPortPair, error) {
	if c.mem.settings != nil && c.mem.settingsByBackend != nil && c.mem.serviceBackendPairsByBackend != nil {
		return *c.mem.settings, *c.mem.settingsByBackend, *c.mem.serviceBackendPairsByBackend, nil
	}

	serviceBackendPairsMap := make(map[backendIdentifier]map[serviceBackendPortPair]interface{})
	backendHTTPSettingsMap := make(map[backendIdentifier]*n.ApplicationGatewayBackendHTTPSettings)
	finalServiceBackendPairMap := make(map[backendIdentifier]serviceBackendPortPair)

	var unresolvedBackendID []backendIdentifier
	for backendID := range c.newBackendIdsFiltered(cbCtx) {
		resolvedBackendPorts := make(map[serviceBackendPortPair]interface{})

		service := c.k8sContext.GetService(backendID.serviceKey())
		if service == nil {
			// This should never happen since newBackendIdsFiltered() already filters out backends for non-existent Services
			logLine := fmt.Sprintf("Unable to get the service [%s]", backendID.serviceKey())
			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonServiceNotFound, logLine)
			glog.Errorf(logLine)
			pair := serviceBackendPortPair{
				ServicePort: Port(backendID.Backend.ServicePort.IntVal),
				BackendPort: Port(backendID.Backend.ServicePort.IntVal),
			}
			resolvedBackendPorts[pair] = nil
		} else {
			for _, sp := range service.Spec.Ports {
				// find the backend port number
				// check if any service ports matches the specified ports
				if sp.Protocol != v1.ProtocolTCP {
					// ignore UDP ports
					continue
				}
				if fmt.Sprint(sp.Port) == backendID.Backend.ServicePort.String() ||
					sp.Name == backendID.Backend.ServicePort.String() ||
					sp.TargetPort.String() == backendID.Backend.ServicePort.String() {
					// matched a service port with a port from the service

					if sp.TargetPort.String() == "" {
						// targetPort is not defined, by default targetPort == port
						pair := serviceBackendPortPair{
							ServicePort: Port(sp.Port),
							BackendPort: Port(sp.Port),
						}
						resolvedBackendPorts[pair] = nil
					} else {
						// target port is defined as name or port number
						if sp.TargetPort.Type == intstr.Int {
							// port is defined as port number
							pair := serviceBackendPortPair{
								ServicePort: Port(sp.Port),
								BackendPort: Port(sp.TargetPort.IntVal),
							}
							resolvedBackendPorts[pair] = nil
						} else {
							// if service port is defined by name, need to resolve
							glog.V(5).Infof("resolving port name [%s] for service [%s] and service port [%s] for Ingress [%s]", sp.Name, backendID.serviceKey(), backendID.Backend.ServicePort.String(), backendID.Ingress.Name)
							targetPortsResolved := c.resolvePortName(sp.Name, &backendID)
							for targetPort := range targetPortsResolved {
								pair := serviceBackendPortPair{
									ServicePort: Port(sp.Port),
									BackendPort: Port(targetPort),
								}
								resolvedBackendPorts[pair] = nil
							}
						}
					}
					break
				}
			}
		}

		if len(resolvedBackendPorts) == 0 {
			logLine := fmt.Sprintf("unable to resolve any backend port for service [%s] and service port [%s] for Ingress [%s]", backendID.serviceKey(), backendID.Backend.ServicePort.String(), backendID.Ingress.Name)
			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonPortResolutionError, logLine)
			glog.Error(logLine)

			unresolvedBackendID = append(unresolvedBackendID, backendID)
			break
		}

		// Merge serviceBackendPairsMap[backendID] into resolvedBackendPorts
		if _, ok := serviceBackendPairsMap[backendID]; !ok {
			serviceBackendPairsMap[backendID] = make(map[serviceBackendPortPair]interface{})
		}
		for portPair := range resolvedBackendPorts {
			serviceBackendPairsMap[backendID][portPair] = nil
		}
	}

	if len(unresolvedBackendID) > 0 {
		return nil, nil, nil, ErrResolvingBackendPortForService
	}

	httpSettingsCollection := make(map[string]n.ApplicationGatewayBackendHTTPSettings)
	defaultBackend := defaultBackendHTTPSettings(c.appGwIdentifier, n.HTTP)
	httpSettingsCollection[*defaultBackend.Name] = defaultBackend

	// enforce single pair relationship between service port and backend port
	for backendID, serviceBackendPairs := range serviceBackendPairsMap {
		if len(serviceBackendPairs) > 1 {
			// more than one possible backend port exposed through ingress
			logLine := fmt.Sprintf("service:port [%s:%s] has more than one service-backend port binding",
				backendID.serviceKey(), backendID.Backend.ServicePort.String())
			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonPortResolutionError, logLine)
			glog.Warning(logLine)
			return nil, nil, nil, ErrMultipleServiceBackendPortBinding
		}

		// At this point there will be only one pair
		var uniquePair serviceBackendPortPair
		for k := range serviceBackendPairs {
			uniquePair = k
		}

		finalServiceBackendPairMap[backendID] = uniquePair
		httpSettings := c.generateHTTPSettings(backendID, uniquePair.BackendPort, cbCtx)
		httpSettingsCollection[*httpSettings.Name] = httpSettings
		backendHTTPSettingsMap[backendID] = &httpSettings
	}

	httpSettings := make([]n.ApplicationGatewayBackendHTTPSettings, 0, len(httpSettingsCollection))
	for _, backend := range httpSettingsCollection {
		httpSettings = append(httpSettings, backend)
	}

	c.mem.settings = &httpSettings
	c.mem.settingsByBackend = &backendHTTPSettingsMap
	c.mem.serviceBackendPairsByBackend = &finalServiceBackendPairMap
	return httpSettings, backendHTTPSettingsMap, finalServiceBackendPairMap, nil
}

func (c *appGwConfigBuilder) generateHTTPSettings(backendID backendIdentifier, port Port, cbCtx *ConfigBuilderContext) n.ApplicationGatewayBackendHTTPSettings {
	httpSettingsName := generateHTTPSettingsName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), port, backendID.Ingress.Name)
	glog.V(5).Infof("Created a new HTTP setting w/ name: %s\n", httpSettingsName)
	httpSettings := n.ApplicationGatewayBackendHTTPSettings{
		Etag: to.StringPtr("*"),
		Name: &httpSettingsName,
		ID:   to.StringPtr(c.appGwIdentifier.HTTPSettingsID(httpSettingsName)),
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Protocol: n.HTTP,
			Port:     to.Int32Ptr(int32(port)),
		},
	}

	_, probesMap := c.newProbesMap(cbCtx)

	if probesMap[backendID] != nil {
		probeName := probesMap[backendID].Name
		probeID := c.appGwIdentifier.probeID(*probeName)
		httpSettings.ApplicationGatewayBackendHTTPSettingsPropertiesFormat.Probe = resourceRef(probeID)
	}

	if pathPrefix, err := annotations.BackendPathPrefix(backendID.Ingress); err == nil {
		httpSettings.Path = to.StringPtr(pathPrefix)
	} else if !errors.IsMissingAnnotations(err) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	if isConnDrain, err := annotations.IsConnectionDraining(backendID.Ingress); err == nil && isConnDrain {
		httpSettings.ConnectionDraining = &n.ApplicationGatewayConnectionDraining{
			Enabled: to.BoolPtr(true),
		}

		if connDrainTimeout, err := annotations.ConnectionDrainingTimeout(backendID.Ingress); err == nil {
			httpSettings.ConnectionDraining.DrainTimeoutInSec = to.Int32Ptr(connDrainTimeout)
		} else {
			httpSettings.ConnectionDraining.DrainTimeoutInSec = to.Int32Ptr(DefaultConnDrainTimeoutInSec)
		}
	} else if err != nil && !errors.IsMissingAnnotations(err) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	if affinity, err := annotations.IsCookieBasedAffinity(backendID.Ingress); err == nil && affinity {
		httpSettings.CookieBasedAffinity = n.Enabled
	} else if err != nil && !errors.IsMissingAnnotations(err) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	if reqTimeout, err := annotations.RequestTimeout(backendID.Ingress); err == nil {
		httpSettings.RequestTimeout = to.Int32Ptr(reqTimeout)
	} else if !errors.IsMissingAnnotations(err) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	if backendProtocol, err := annotations.BackendProtocol(backendID.Ingress); err == nil && backendProtocol == annotations.HTTPS {
		httpSettings.Protocol = n.HTTPS
	} else if err != nil && !errors.IsMissingAnnotations(err) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	return httpSettings
}
