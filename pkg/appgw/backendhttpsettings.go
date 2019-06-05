// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"errors"
	"fmt"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// DefaultConnDrainTimeoutInSec provides default value for ConnectionDrainTimeout
	DefaultConnDrainTimeoutInSec = 30
)

func newBackendIds(ingressList []*v1beta1.Ingress) map[backendIdentifier]interface{} {
	backendIDs := make(map[backendIdentifier]interface{})
	for _, ingress := range ingressList {
		if ingress.Spec.Backend != nil {
			glog.Infof("Ingress spec has no backend. Adding a default.")
			backendIDs[generateBackendID(ingress, nil, nil, ingress.Spec.Backend)] = nil
		}
		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			glog.Infof("Working on ingress rule #%d: host='%s'", ruleIdx+1, rule.Host)
			if rule.HTTP == nil {
				// skip no http rule
				glog.Infof("Skip rule #%d for host '%s' - it has no HTTP rules.", ruleIdx+1, rule.Host)
				continue
			}
			for pathIdx := range rule.HTTP.Paths {
				path := &rule.HTTP.Paths[pathIdx]
				glog.Infof("Working on path #%d: '%s'", pathIdx+1, path.Path)
				backendIDs[generateBackendID(ingress, rule, path, &path.Backend)] = nil
			}
		}
	}
	return backendIDs
}

func (c *appGwConfigBuilder) BackendHTTPSettingsCollection(ingressList []*v1beta1.Ingress) error {
	serviceBackendPairsMap := make(map[backendIdentifier]map[serviceBackendPortPair]interface{})
	backendIDs := newBackendIds(ingressList)

	var unresolvedBackendID []backendIdentifier
	for backendID := range backendIDs {
		resolvedBackendPorts := make(map[serviceBackendPortPair]interface{})

		service := c.k8sContext.GetService(backendID.serviceKey())
		if service == nil {
			logLine := fmt.Sprintf("Unable to get the service [%s]", backendID.serviceKey())
			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, "ServiceNotFound", logLine)
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
			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, "PortResolutionError", logLine)
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
		return errors.New("unable to resolve backend port for some services")
	}

	probeID := c.appGwIdentifier.probeID(defaultProbeName)
	httpSettingsCollection := make(map[string]network.ApplicationGatewayBackendHTTPSettings)
	defaultBackend := defaultBackendHTTPSettings(probeID)
	httpSettingsCollection[*defaultBackend.Name] = defaultBackend

	// enforce single pair relationship between service port and backend port
	for backendID, serviceBackendPairs := range serviceBackendPairsMap {
		if len(serviceBackendPairs) > 1 {
			// more than one possible backend port exposed through ingress
			logLine := fmt.Sprintf("service:port [%s:%s] has more than one service-backend port binding",
				backendID.serviceKey(), backendID.Backend.ServicePort.String())
			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, "PortResolutionError", logLine)
			glog.Warning(logLine)
			return errors.New("more than one service-backend port binding is not allowed")
		}

		// At this point there will be only one pair
		var uniquePair serviceBackendPortPair
		for k := range serviceBackendPairs {
			uniquePair = k
		}

		c.serviceBackendPairMap[backendID] = uniquePair
		httpSettings := c.generateHTTPSettings(backendID, uniquePair.BackendPort)
		httpSettingsCollection[*httpSettings.Name] = httpSettings
		c.backendHTTPSettingsMap[backendID] = &httpSettings
	}

	backends := make([]network.ApplicationGatewayBackendHTTPSettings, 0, len(httpSettingsCollection))
	for _, backend := range httpSettingsCollection {
		backends = append(backends, backend)
	}

	c.appGwConfig.BackendHTTPSettingsCollection = &backends

	return nil
}

func (c *appGwConfigBuilder) generateHTTPSettings(backendID backendIdentifier, port int32) network.ApplicationGatewayBackendHTTPSettings {
	probeName := c.probesMap[backendID].Name
	probeID := c.appGwIdentifier.probeID(*probeName)
	httpSettingsName := generateHTTPSettingsName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), port, backendID.Ingress.Name)
	glog.Infof("Created a new HTTP setting w/ name: %s\n", httpSettingsName)

	httpSettings := network.ApplicationGatewayBackendHTTPSettings{
		Etag: to.StringPtr("*"),
		Name: &httpSettingsName,
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &network.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Protocol: network.HTTP,
			Port:     &port,
			Probe:    resourceRef(probeID),
		},
	}

	if pathPrefix, err := annotations.BackendPathPrefix(backendID.Ingress); err == nil {
		httpSettings.Path = to.StringPtr(pathPrefix)
	}

	if isConnDrain, err := annotations.IsConnectionDraining(backendID.Ingress); err == nil && isConnDrain {
		httpSettings.ConnectionDraining = &network.ApplicationGatewayConnectionDraining{
			Enabled: to.BoolPtr(true),
		}

		if connDrainTimeout, err := annotations.ConnectionDrainingTimeout(backendID.Ingress); err == nil {
			httpSettings.ConnectionDraining.DrainTimeoutInSec = to.Int32Ptr(connDrainTimeout)
		} else {
			httpSettings.ConnectionDraining.DrainTimeoutInSec = to.Int32Ptr(DefaultConnDrainTimeoutInSec)
		}
	}

	if affinity, err := annotations.IsCookieBasedAffinity(backendID.Ingress); err == nil && affinity {
		httpSettings.CookieBasedAffinity = network.Enabled
	}

	if reqTimeout, err := annotations.RequestTimeout(backendID.Ingress); err == nil {
		httpSettings.RequestTimeout = to.Int32Ptr(reqTimeout)
	}

	return httpSettings
}
