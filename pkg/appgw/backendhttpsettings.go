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
		rCtx := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)

		// PathMaps we obtained from App Gateway - we segment them into ones AGIC is and is not allowed to change.
		existingBlacklisted, existingNonBlacklisted := rCtx.GetBlacklistedHTTPSettings()

		brownfield.LogHTTPSettings(klog.V(3), existingBlacklisted, existingNonBlacklisted, agicHTTPSettings)

		// MergePathMaps would produce unique list of routing rules based on Name. Routing rules, which have the same name
		// as a managed rule would be overwritten.
		agicHTTPSettings = brownfield.MergeHTTPSettings(existingBlacklisted, agicHTTPSettings)
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

	defaultBackend := defaultBackendHTTPSettings(c.appGwIdentifier, n.HTTP)
	serviceBackendPairMap := make(map[backendIdentifier]serviceBackendPortPair)
	backendHTTPSettingsMap := make(map[backendIdentifier]*n.ApplicationGatewayBackendHTTPSettings)
	httpSettingsCollection := make(map[string]n.ApplicationGatewayBackendHTTPSettings)
	httpSettingsCollection[*defaultBackend.Name] = defaultBackend
	for backendID := range c.newBackendIdsFiltered(cbCtx) {
		backendPort, err := c.resolveBackendPort(backendID)
		if err != nil {
			c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonPortResolutionError, err.Error())
			klog.Error(err.Error())
		}

		httpSettings := c.generateHTTPSettings(backendID, backendPort, cbCtx)
		klog.V(5).Infof("Created backend http settings %s for ingress %s/%s and service %s", *httpSettings.Name, backendID.Ingress.Namespace, backendID.Ingress.Name, backendID.serviceKey())

		// TODO(aksgupta): Only backend port is used in the output; remove service port.
		serviceBackendPairMap[backendID] = serviceBackendPortPair{
			ServicePort: backendPort,
			BackendPort: backendPort,
		}

		httpSettingsCollection[*httpSettings.Name] = httpSettings
		backendHTTPSettingsMap[backendID] = &httpSettings
	}

	httpSettings := make([]n.ApplicationGatewayBackendHTTPSettings, 0, len(httpSettingsCollection))
	for _, backend := range httpSettingsCollection {
		httpSettings = append(httpSettings, backend)
	}

	c.mem.settings = &httpSettings
	c.mem.settingsByBackend = &backendHTTPSettingsMap
	c.mem.serviceBackendPairsByBackend = &serviceBackendPairMap
	return httpSettings, backendHTTPSettingsMap, serviceBackendPairMap, nil
}

func (c *appGwConfigBuilder) resolveBackendPort(backendID backendIdentifier) (Port, error) {
	var e error
	service := c.k8sContext.GetService(backendID.serviceKey())
	if service == nil {
		// This should never happen since newBackendIdsFiltered() already filters out backends for non-existent Services
		e = controllererrors.NewErrorf(
			controllererrors.ErrorServiceNotFound,
			"Service not found %s",
			backendID.serviceKey())
		return Port(backendID.Backend.ServicePort.IntVal), e
	}

	// find the target port number for service port specified in the ingress manifest
	servicePortInIngress := backendID.Backend.ServicePort.String()
	resolvedBackendPorts := make(map[serviceBackendPortPair]interface{})
	for _, servicePort := range service.Spec.Ports {
		// ignore UDP ports
		if servicePort.Protocol != v1.ProtocolTCP {
			continue
		}

		// match service by either port, port name or target port
		if fmt.Sprint(servicePort.Port) != servicePortInIngress &&
			servicePort.Name != servicePortInIngress &&
			servicePort.TargetPort.String() != servicePortInIngress {
			continue
		}

		// if target port is not specified, use the service port as backend port
		if servicePort.TargetPort.String() == "" {
			// targetPort is not defined, by default targetPort == port
			pair := serviceBackendPortPair{
				ServicePort: Port(servicePort.Port),
				BackendPort: Port(servicePort.Port),
			}
			resolvedBackendPorts[pair] = nil
			continue
		}

		// if target port is an int, use it as backend port
		if servicePort.TargetPort.Type == intstr.Int {
			// port is defined as port number
			pair := serviceBackendPortPair{
				ServicePort: Port(servicePort.Port),
				BackendPort: Port(servicePort.TargetPort.IntVal),
			}
			resolvedBackendPorts[pair] = nil
			continue
		}

		// if target port is port name, then resolve the port number for the port name
		// k8s matches service port name against endpoints port name retrieved by passing backendID service key to endpoint api.
		klog.V(5).Infof("resolving port name '%s' for service '%s' and service port '%s' for Ingress '%s'", servicePort.Name, backendID.serviceKey(), backendID.Backend.ServicePort.String(), backendID.Ingress.Name)
		targetPortsResolved := c.resolvePortName(servicePort.Name, &backendID)
		for targetPort := range targetPortsResolved {
			pair := serviceBackendPortPair{
				ServicePort: Port(servicePort.Port),
				BackendPort: Port(targetPort),
			}
			resolvedBackendPorts[pair] = nil
		}
	}

	if len(resolvedBackendPorts) == 0 {
		e = controllererrors.NewErrorf(
			controllererrors.ErrorUnableToResolveBackendPortFromServicePort,
			"No port matched %s",
			backendID.serviceKey())

		// if service port is an int, use that as backend port
		if backendID.Backend.ServicePort.Type == intstr.Int {
			return Port(backendID.Backend.ServicePort.IntVal), e
		}

		// returning port 80 as a last resort
		return Port(80), e
	}

	backendPort := Port(65536)
	var ports []string
	for k := range resolvedBackendPorts {
		ports = append(ports, string(k.BackendPort))
		if k.BackendPort <= backendPort {
			backendPort = k.BackendPort
		}
	}

	if len(resolvedBackendPorts) > 1 {
		// found more than 1 backend port for the service port which is a conflicting scenario
		e = controllererrors.NewErrorf(
			controllererrors.ErrorMultipleServiceBackendPortBinding,
			"service:port [%s:%s] has more than one service-backend port binding which is not an ideal scenario, choosing the smallest service-backend port %d. Ports found %s.",
			backendID.serviceKey(), backendID.Backend.ServicePort.String(), backendPort, strings.Join(ports, ","),
		)
	}

	return backendPort, e
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
