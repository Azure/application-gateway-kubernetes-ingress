// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"errors"
	"fmt"
	"sort"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

const (
	// DefaultConnDrainTimeoutInSec provides default value for ConnectionDrainTimeout
	DefaultConnDrainTimeoutInSec = 30
)

func (c *appGwConfigBuilder) BackendHTTPSettingsCollection(cbCtx *ConfigBuilderContext) error {
	agicHTTPSettings, _, _, err := c.getBackendsAndSettingsMap(cbCtx)
	if agicHTTPSettings != nil {
		sort.Sort(sorter.BySettingsName(*agicHTTPSettings))
	}
	c.appGw.BackendHTTPSettingsCollection = agicHTTPSettings
	return err
}

func newBackendIdsFiltered(cbCtx *ConfigBuilderContext) map[backendIdentifier]interface{} {
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
	return finalBackendIDs
}

func istioBackendIds(cbCtx *ConfigBuilderContext) []istioBackendIdentifier {
	backendIDs := make([]istioBackendIdentifier, 0)
	for _, virtualService := range cbCtx.IstioVirtualServices {
		for _, rule := range virtualService.Spec.HTTP {
			destinations := make([]*v1alpha3.Destination, 0)
			for _, routeDestination := range rule.Route {
				if routeDestination.Weight != 0 {
					destinations = append(destinations, &routeDestination.Destination)
					/* TODO(rhea): Weights are being ignored for now, since this is not
					yet supported on App Gateway. Include gates from routeDestination when
					this is supported */
				}
			}
			for _, match := range rule.Match {
				if match.URI == nil {
					glog.V(5).Infof("Skipped match request, no URI field. Other forms of match requests are not supported.")
					continue
				}
				backendID := generateIstioBackendID(virtualService, &rule, &match, destinations)
				backendIDs = append(backendIDs, backendID)
			}
		}
	}
	/* TODO(rhea): Filter out backends for virtual services referencing non-existent Services */
	return backendIDs
}

func newServiceSet(services *[]*v1.Service) map[string]*v1.Service {
	servicesSet := make(map[string]*v1.Service)
	for _, service := range *services {
		serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
		servicesSet[serviceKey] = service
	}
	return servicesSet
}

func (c *appGwConfigBuilder) getBackendsAndSettingsMap(cbCtx *ConfigBuilderContext) (*[]n.ApplicationGatewayBackendHTTPSettings, map[backendIdentifier]*n.ApplicationGatewayBackendHTTPSettings, map[backendIdentifier]serviceBackendPortPair, error) {
	serviceBackendPairsMap := make(map[backendIdentifier]map[serviceBackendPortPair]interface{})
	backendHTTPSettingsMap := make(map[backendIdentifier]*n.ApplicationGatewayBackendHTTPSettings)
	finalServiceBackendPairMap := make(map[backendIdentifier]serviceBackendPortPair)

	var unresolvedBackendID []backendIdentifier
	for backendID := range newBackendIdsFiltered(cbCtx) {
		resolvedBackendPorts := make(map[serviceBackendPortPair]interface{})

		service := c.k8sContext.GetService(backendID.serviceKey())
		if service == nil {
			// This should never happen since newBackendIdsFiltered() already filters out backends for non-existent Services
			logLine := fmt.Sprintf("Unable to get the service [%s]", backendID.serviceKey())
			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonServiceNotFound, logLine)
			glog.Errorf(logLine)
			pair := serviceBackendPortPair{
				ServicePort: backendID.Backend.ServicePort.IntVal,
				BackendPort: backendID.Backend.ServicePort.IntVal,
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
							ServicePort: sp.Port,
							BackendPort: sp.Port,
						}
						resolvedBackendPorts[pair] = nil
					} else {
						// target port is defined as name or port number
						if sp.TargetPort.Type == intstr.Int {
							// port is defined as port number
							pair := serviceBackendPortPair{
								ServicePort: sp.Port,
								BackendPort: sp.TargetPort.IntVal,
							}
							resolvedBackendPorts[pair] = nil
						} else {
							// if service port is defined by name, need to resolve
							targetPortName := sp.TargetPort.StrVal
							glog.V(1).Infof("resolving port name %s", targetPortName)
							targetPortsResolved := c.resolvePortName(targetPortName, &backendID)
							for targetPort := range targetPortsResolved {
								pair := serviceBackendPortPair{
									ServicePort: sp.Port,
									BackendPort: targetPort,
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
			logLine := fmt.Sprintf("Unable to resolve any backend port for service [%s]", backendID.serviceKey())
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
		return nil, nil, nil, errors.New("unable to resolve backend port for some services")
	}

	probeID := c.appGwIdentifier.probeID(defaultProbeName)
	httpSettingsCollection := make(map[string]n.ApplicationGatewayBackendHTTPSettings)
	defaultBackend := defaultBackendHTTPSettings(probeID)
	httpSettingsCollection[*defaultBackend.Name] = defaultBackend

	// enforce single pair relationship between service port and backend port
	for backendID, serviceBackendPairs := range serviceBackendPairsMap {
		if len(serviceBackendPairs) > 1 {
			// more than one possible backend port exposed through ingress
			logLine := fmt.Sprintf("service:port [%s:%s] has more than one service-backend port binding",
				backendID.serviceKey(), backendID.Backend.ServicePort.String())
			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonPortResolutionError, logLine)
			glog.Warning(logLine)
			return nil, nil, nil, errors.New("more than one service-backend port binding is not allowed")
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

	return &httpSettings, backendHTTPSettingsMap, finalServiceBackendPairMap, nil
}

func (c *appGwConfigBuilder) generateHTTPSettings(backendID backendIdentifier, port int32, cbCtx *ConfigBuilderContext) n.ApplicationGatewayBackendHTTPSettings {
	httpSettingsName := generateHTTPSettingsName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), port, backendID.Ingress.Name)
	glog.V(5).Infof("Created a new HTTP setting w/ name: %s\n", httpSettingsName)
	httpSettings := n.ApplicationGatewayBackendHTTPSettings{
		Etag: to.StringPtr("*"),
		Name: &httpSettingsName,
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Protocol: n.HTTP,
			Port:     &port,
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
	}

	if affinity, err := annotations.IsCookieBasedAffinity(backendID.Ingress); err == nil && affinity {
		httpSettings.CookieBasedAffinity = n.Enabled
	}

	if reqTimeout, err := annotations.RequestTimeout(backendID.Ingress); err == nil {
		httpSettings.RequestTimeout = to.Int32Ptr(reqTimeout)
	}

	return httpSettings
}
