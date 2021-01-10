// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"sort"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
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
		rCtx := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, cbCtx.AllowedTargets, nil)

		// PathMaps we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
		var existingNonAllowed []n.ApplicationGatewayBackendHTTPSettings
		var existingAllowed []n.ApplicationGatewayBackendHTTPSettings

		if cbCtx.EnvVariables.UseAllowedTargetsBrownfieldDeployment {
			existingNonAllowed, existingAllowed = rCtx.GetWhitelistedHTTPSettings()
		} else {
			existingNonAllowed, existingAllowed = rCtx.GetBlacklistedHTTPSettings()
		}

		brownfield.LogHTTPSettings(klog.V(3), existingNonAllowed, existingAllowed, agicHTTPSettings)

		// MergePathMaps would produce unique list of routing rules based on Name. Routing rules, which have the same name
		// as a managed rule would be overwritten.
		agicHTTPSettings = brownfield.MergeHTTPSettings(existingNonAllowed, agicHTTPSettings)
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
			klog.Errorf(logLine)
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
							klog.V(5).Infof("resolving port name [%s] for service [%s] and service port [%s] for Ingress [%s]", sp.Name, backendID.serviceKey(), backendID.Backend.ServicePort.String(), backendID.Ingress.Name)

							// k8s matches service port name against endpoints port name retrieved by passing backendID service key to endpoint api.
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
			klog.Error(logLine)

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
		klog.Warningf("Unable to resolve %d backends: %+v", len(unresolvedBackendID), unresolvedBackendID)
	}

	httpSettingsCollection := make(map[string]n.ApplicationGatewayBackendHTTPSettings)
	defaultBackend := defaultBackendHTTPSettings(c.appGwIdentifier, n.HTTP)
	httpSettingsCollection[*defaultBackend.Name] = defaultBackend

	// enforce single pair relationship between service port and backend port
	for backendID, serviceBackendPairs := range serviceBackendPairsMap {

		// in case there are multiple backend ports found, using the smallest port in http setting
		var backendport Port
		backendport = 65536

		// this will store all the ports found
		var ports []string

		var uniquePair serviceBackendPortPair
		for k := range serviceBackendPairs {
			ports = append(ports, fmt.Sprintf("%d", k.BackendPort))
			if k.BackendPort <= backendport {
				uniquePair = k
				backendport = k.BackendPort
			}
		}

		if len(serviceBackendPairs) > 1 {

			// more than one possible backend port exposed through ingress
			e := controllererrors.NewErrorf(
				controllererrors.ErrorMultipleServiceBackendPortBinding,
				"service:port [%s:%s] has more than one service-backend port binding which is not an ideal scenario, choosing the smallest service-backend port %d. Ports found %s.",
				backendID.serviceKey(), backendID.Backend.ServicePort.String(), backendport, strings.Join(ports, ","),
			)

			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonPortResolutionError, e.Error())
			klog.Errorf(e.Error())
		}

		finalServiceBackendPairMap[backendID] = uniquePair
		httpSettings := c.generateHTTPSettings(backendID, uniquePair.BackendPort, cbCtx)
		klog.V(5).Infof("Created backend http settings %s for ingress %s/%s and service %s", *httpSettings.Name, backendID.Ingress.Namespace, backendID.Ingress.Name, backendID.serviceKey())
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

	httpSettings := n.ApplicationGatewayBackendHTTPSettings{
		Etag: to.StringPtr("*"),
		Name: &httpSettingsName,
		ID:   to.StringPtr(c.appGwIdentifier.HTTPSettingsID(httpSettingsName)),
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Protocol: n.HTTP,
			Port:     to.Int32Ptr(int32(port)),

			// setting to default
			PickHostNameFromBackendAddress: to.BoolPtr(false),
			CookieBasedAffinity:            n.Disabled,
			RequestTimeout:                 to.Int32Ptr(30),
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
	} else if !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	if hostName, err := annotations.BackendHostName(backendID.Ingress); err == nil {
		httpSettings.HostName = to.StringPtr(hostName)
	} else if !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
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
	} else if err != nil && !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	if affinity, err := annotations.IsCookieBasedAffinity(backendID.Ingress); err == nil && affinity {
		httpSettings.CookieBasedAffinity = n.Enabled
	} else if err != nil && !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	if reqTimeout, err := annotations.RequestTimeout(backendID.Ingress); err == nil {
		httpSettings.RequestTimeout = to.Int32Ptr(reqTimeout)
	} else if !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	// when ingress is defined with backend at port 443 but without annotation backend-protocol set to https.
	if int32(port) == 443 {
		httpSettings.Protocol = n.HTTPS
	}

	// backend protocol take precedence over port
	backendProtocol, err := annotations.BackendProtocol(backendID.Ingress)
	if err == nil && backendProtocol == annotations.HTTPS {
		httpSettings.Protocol = n.HTTPS
	} else if err == nil && backendProtocol == annotations.HTTP {
		httpSettings.Protocol = n.HTTP
	} else if err != nil && !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	if trustedRootCertificates, err := annotations.GetAppGwTrustedRootCertificate(backendID.Ingress); err == nil {
		certificateNames := strings.TrimRight(trustedRootCertificates, ",")
		certificateNameList := strings.Split(certificateNames, ",")
		var certs []n.SubResource
		for _, certName := range certificateNameList {
			trustCertID := c.appGwIdentifier.trustedRootCertificateID(certName)
			certs = append(certs, *resourceRef(trustCertID))
		}
		httpSettings.TrustedRootCertificates = &certs
		klog.V(5).Infof("Found trusted root certificate(s): %s from ingress: %s/%s", certificateNames, backendID.Ingress.Namespace, backendID.Ingress.Name)

	} else if err != nil && !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonInvalidAnnotation, err.Error())
	}

	// To use an HTTP setting with a trusted root certificate, we must either override with a specific domain name or choose "Pick host name from backend target".
	if httpSettings.TrustedRootCertificates != nil {
		if httpSettings.Protocol == n.HTTPS && len(*httpSettings.TrustedRootCertificates) > 0 {
			if httpSettings.HostName != nil && len(*httpSettings.HostName) > 0 {
				httpSettings.PickHostNameFromBackendAddress = to.BoolPtr(false)
			} else {
				httpSettings.PickHostNameFromBackendAddress = to.BoolPtr(true)
			}
		}
	}

	return httpSettings
}
