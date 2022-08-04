// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"sort"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

func (c *appGwConfigBuilder) HealthProbesCollection(cbCtx *ConfigBuilderContext) error {
	healthProbeCollection, _ := c.newProbesMap(cbCtx)
	agicCreatedProbes := make([]n.ApplicationGatewayProbe, 0, len(healthProbeCollection))
	for _, probe := range healthProbeCollection {
		agicCreatedProbes = append(agicCreatedProbes, probe)
	}

	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		er := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)
		existingBlacklisted, existingNonBlacklisted := er.GetBlacklistedProbes()
		brownfield.LogProbes(klog.V(3), existingBlacklisted, existingNonBlacklisted, agicCreatedProbes)
		agicCreatedProbes = brownfield.MergeProbes(existingBlacklisted, agicCreatedProbes)
	}

	sort.Sort(sorter.ByHealthProbeName(agicCreatedProbes))
	c.appGw.Probes = &agicCreatedProbes
	return nil
}

func (c *appGwConfigBuilder) newProbesMap(cbCtx *ConfigBuilderContext) (map[string]n.ApplicationGatewayProbe, map[backendIdentifier]*n.ApplicationGatewayProbe) {
	if c.mem.probesByName != nil && c.mem.probesByBackend != nil {
		return *c.mem.probesByName, *c.mem.probesByBackend
	}

	healthProbeCollection := make(map[string]n.ApplicationGatewayProbe)
	probesMap := make(map[backendIdentifier]*n.ApplicationGatewayProbe)
	defaultHTTPProbe := defaultProbe(c.appGwIdentifier, n.ApplicationGatewayProtocolHTTP)
	defaultHTTPSProbe := defaultProbe(c.appGwIdentifier, n.ApplicationGatewayProtocolHTTPS)

	healthProbeCollection[*defaultHTTPProbe.Name] = defaultHTTPProbe
	healthProbeCollection[*defaultHTTPSProbe.Name] = defaultHTTPSProbe
	klog.V(5).Info("Created default HTTP probe ", *defaultHTTPProbe.Name)
	klog.V(5).Info("Created default HTTPS probe ", *defaultHTTPProbe.Name)

	for backendID := range c.newBackendIdsFiltered(cbCtx) {
		probe := c.generateHealthProbe(backendID)

		if probe != nil {
			probesMap[backendID] = probe
			healthProbeCollection[*probe.Name] = *probe
		} else {
			probesMap[backendID] = &defaultHTTPProbe
			if protocol, _ := annotations.BackendProtocol(backendID.Ingress); protocol == annotations.HTTPS {
				probesMap[backendID] = &defaultHTTPSProbe
			}
		}
		klog.V(5).Infof("Created probe %s for ingress %s/%s at service %s", *probesMap[backendID].Name, backendID.Ingress.Namespace, backendID.Ingress.Name, backendID.serviceKey())
	}

	c.mem.probesByName = &healthProbeCollection
	c.mem.probesByBackend = &probesMap
	return healthProbeCollection, probesMap
}

func (c *appGwConfigBuilder) generateHealthProbe(backendID backendIdentifier) *n.ApplicationGatewayProbe {
	// TODO(draychev): remove GetService
	service := c.k8sContext.GetService(backendID.serviceKey())
	if service == nil || backendID.Path == nil {
		return nil
	}
	probe := defaultProbe(c.appGwIdentifier, n.ApplicationGatewayProtocolHTTP)
	probe.Name = to.StringPtr(generateProbeName(backendID.Path.Backend.Service.Name, serviceBackendPortToStr(backendID.Path.Backend.Service.Port), backendID.Ingress))
	probe.ID = to.StringPtr(c.appGwIdentifier.probeID(*probe.Name))

	// set defaults
	probe.Match = &n.ApplicationGatewayProbeHealthResponseMatch{}
	probe.PickHostNameFromBackendHTTPSettings = to.BoolPtr(false)
	probe.MinServers = to.Int32Ptr(0)

	listenerID := generateListenerID(backendID.Ingress, backendID.Rule, n.ApplicationGatewayProtocolHTTP, nil, false)
	hostName := listenerID.getHostNameForProbes()
	if hostName != nil {
		probe.Host = hostName
	}

	if hostName, err := annotations.BackendHostName(backendID.Ingress); err == nil {
		probe.Host = to.StringPtr(hostName)
	}

	pathPrefix, err := annotations.BackendPathPrefix(backendID.Ingress)
	if err == nil {
		probe.Path = to.StringPtr(pathPrefix)
	} else if backendID.Path != nil && len(backendID.Path.Path) != 0 {
		probe.Path = to.StringPtr(backendID.Path.Path)
	}

	// check if backend is using port 443
	if backendID.Backend.Service != nil {
		if serviceBackendPortToStr(backendID.Backend.Service.Port) == "443" {
			probe.Protocol = n.ApplicationGatewayProtocolHTTPS
		} else if port, err := c.resolveBackendPort(backendID); err == nil && port == Port(443) {
			probe.Protocol = n.ApplicationGatewayProtocolHTTPS
		}
	}

	k8sProbeForServiceContainer := c.getProbeForServiceContainer(service, backendID)
	if k8sProbeForServiceContainer != nil {
		if len(k8sProbeForServiceContainer.Handler.HTTPGet.Host) != 0 {
			probe.Host = to.StringPtr(k8sProbeForServiceContainer.Handler.HTTPGet.Host)
		}
		if len(k8sProbeForServiceContainer.Handler.HTTPGet.Path) != 0 {
			probe.Path = to.StringPtr(k8sProbeForServiceContainer.Handler.HTTPGet.Path)
		}
		if len(k8sProbeForServiceContainer.Handler.HTTPGet.Port.String()) != 0 {
			probe.Port = to.Int32Ptr(k8sProbeForServiceContainer.Handler.HTTPGet.Port.IntVal)
		}
		if k8sProbeForServiceContainer.Handler.HTTPGet.Scheme == v1.URISchemeHTTPS {
			probe.Protocol = n.ApplicationGatewayProtocolHTTPS
		}
		// httpGet schema is default to Http if not specified, double check with the port in case for Https
		if k8sProbeForServiceContainer.Handler.HTTPGet.Scheme == v1.URISchemeHTTP {
			if k8sProbeForServiceContainer.Handler.HTTPGet.Port.IntVal == 443 {
				probe.Protocol = n.ApplicationGatewayProtocolHTTPS
			} else {
				probe.Protocol = n.ApplicationGatewayProtocolHTTP
			}
		}
		if k8sProbeForServiceContainer.PeriodSeconds != 0 {
			probe.Interval = to.Int32Ptr(k8sProbeForServiceContainer.PeriodSeconds)
		}
		if k8sProbeForServiceContainer.TimeoutSeconds != 0 {
			probe.Timeout = to.Int32Ptr(k8sProbeForServiceContainer.TimeoutSeconds)
		}
		if k8sProbeForServiceContainer.FailureThreshold != 0 {
			// UnhealthyThreshold must be in range of 0 - 20, otherwise Application Gateway will not
			// accept the configuration and all new pods will not be configured.
			if k8sProbeForServiceContainer.FailureThreshold > 20 {
				probe.UnhealthyThreshold = to.Int32Ptr(20)
			} else {
				probe.UnhealthyThreshold = to.Int32Ptr(k8sProbeForServiceContainer.FailureThreshold)
			}
		}
	}

	// backend protocol must match http settings protocol
	backendProtocol, err := annotations.BackendProtocol(backendID.Ingress)
	if err == nil && backendProtocol == annotations.HTTPS {
		probe.Protocol = n.ApplicationGatewayProtocolHTTPS
	} else if err == nil && backendProtocol == annotations.HTTP {
		probe.Protocol = n.ApplicationGatewayProtocolHTTP
	}

	// override healthcheck probe host with host defined in annotation if exists
	probeHost, err := annotations.HealthProbeHostName(backendID.Ingress)
	if err == nil && probeHost != "" {
		probe.Host = to.StringPtr(probeHost)
	}

	// override healthcheck probe target port with port defined in annotation if exists
	probePort, err := annotations.HealthProbePort(backendID.Ingress)
	if err == nil && probePort > 0 && probePort < 65536 {
		probe.Port = to.Int32Ptr(probePort)
	}

	// override healthcheck probe path with path defined in annotation if exists
	probePath, err := annotations.HealthProbePath(backendID.Ingress)
	if err == nil && probePath != "" {
		probe.Path = to.StringPtr(probePath)
	}

	if probe.Path != nil {
		probe.Path = to.StringPtr(strings.TrimRight(*probe.Path, "*"))
	}

	// override healthcheck probe match status codes with ones defined in annotation if exists
	probeStatuses, err := annotations.HealthProbeStatusCodes(backendID.Ingress)
	if err == nil && len(probeStatuses) > 0 {
		probe.Match.StatusCodes = &probeStatuses
	}

	// override healthcheck probe interval with value defined in annotation if exists
	probeInterval, err := annotations.HealthProbeInterval(backendID.Ingress)
	if err == nil && probeInterval > 0 {
		probe.Interval = to.Int32Ptr(probeInterval)
	}

	// override healthcheck probe timeout with value defined in annotation if exists
	probeTimeout, err := annotations.HealthProbeTimeout(backendID.Ingress)
	if err == nil && probeTimeout > 0 {
		probe.Timeout = to.Int32Ptr(probeTimeout)
	}

	// override healthcheck probe threshold with value defined in annotation if exists
	probeThreshold, err := annotations.HealthProbeUnhealthyThreshold(backendID.Ingress)
	if err == nil && probeThreshold > 0 {
		probe.UnhealthyThreshold = to.Int32Ptr(probeThreshold)
	}

	// For V1 gateway, port property is not supported
	if c.appGw.Sku.Tier == n.ApplicationGatewayTierStandard || c.appGw.Sku.Tier == n.ApplicationGatewayTierWAF {
		probe.Port = nil
	}

	return &probe
}

func (c *appGwConfigBuilder) getProbeForServiceContainer(service *v1.Service, backendID backendIdentifier) *v1.Probe {
	// find all the target ports used by the service
	allPorts := make(map[int32]interface{})
	for _, sp := range service.Spec.Ports {
		if sp.Protocol != v1.ProtocolTCP {
			continue
		}

		if backendID.Backend.Service != nil && (fmt.Sprint(sp.Port) == serviceBackendPortToStr(backendID.Backend.Service.Port) ||
			sp.Name == serviceBackendPortToStr(backendID.Backend.Service.Port) ||
			sp.TargetPort.String() == serviceBackendPortToStr(backendID.Backend.Service.Port)) {

			// Matched a service port in the service
			if sp.TargetPort.String() == "" {
				allPorts[sp.Port] = nil
			} else if sp.TargetPort.Type == intstr.Int {
				// port is defined as port number
				allPorts[sp.TargetPort.IntVal] = nil
			} else {
				for targetPort := range c.resolvePortName(sp.Name, &backendID) {
					allPorts[targetPort] = nil
				}
			}
		}
	}

	podList := c.k8sContext.ListPodsByServiceSelector(service)

	if len(podList) == 0 {
		return nil
	}

	// use the target port to figure out the container and use it's readiness/liveness probe
	for _, container := range podList[0].Spec.Containers {
		for _, port := range container.Ports {
			if _, ok := allPorts[port.ContainerPort]; !ok {
				continue
			}

			// found the container
			var probe *v1.Probe
			if container.ReadinessProbe != nil && container.ReadinessProbe.Handler.HTTPGet != nil {
				probe = container.ReadinessProbe
			} else if container.LivenessProbe != nil && container.LivenessProbe.Handler.HTTPGet != nil {
				probe = container.LivenessProbe
			}

			// if probe port is named, resolve it by going through container port and set it in the probe itself
			if probe != nil && probe.HTTPGet.Port.String() != "" && probe.HTTPGet.Port.Type == intstr.String {
				for _, port := range container.Ports {
					if port.Name == probe.HTTPGet.Port.StrVal {
						probe.HTTPGet.Port = intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: port.ContainerPort,
						}
						break
					}
				}
			}

			return probe
		}
	}

	return nil
}
