// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/record"
)

// ConfigBuilder is a builder for application gateway configuration
type ConfigBuilder interface {
	// builder pattern
	BackendHTTPSettingsCollection(ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error
	BackendAddressPools(ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error
	Listeners(ingressList []*v1beta1.Ingress) error
	RequestRoutingRules(ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error
	HealthProbesCollection(ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error
	GetApplicationGatewayPropertiesFormatPtr() *network.ApplicationGatewayPropertiesFormat
	PreBuildValidate(envVariables environment.EnvVariables, ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error
	PostBuildValidate(envVariables environment.EnvVariables, ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error
}

type appGwConfigBuilder struct {
	k8sContext      *k8scontext.Context
	appGwIdentifier Identifier
	appGwConfig     network.ApplicationGatewayPropertiesFormat
	recorder        record.EventRecorder
}

// NewConfigBuilder construct a builder
func NewConfigBuilder(context *k8scontext.Context, appGwIdentifier *Identifier, originalConfig *network.ApplicationGatewayPropertiesFormat, recorder record.EventRecorder) ConfigBuilder {
	return &appGwConfigBuilder{
		// TODO(draychev): Decommission internal state
		k8sContext:      context,
		appGwIdentifier: *appGwIdentifier,
		appGwConfig:     *originalConfig,
		recorder:        recorder,
	}
}

// resolvePortName function goes through the endpoints of a given service and
// look for possible port number corresponding to a port name
func (c *appGwConfigBuilder) resolvePortName(portName string, backendID *backendIdentifier) map[int32]interface{} {
	resolvedPorts := make(map[int32]interface{})
	endpoints, err := c.k8sContext.GetEndpointsByService(backendID.serviceKey())
	if err != nil {
		glog.Error("Could not fetch endpoint by service key from cache", err)
		return resolvedPorts
	}

	if endpoints == nil {
		return resolvedPorts
	}
	for _, subset := range endpoints.Subsets {
		for _, epPort := range subset.Ports {
			if epPort.Name == portName {
				resolvedPorts[epPort.Port] = nil
			}
		}
	}
	return resolvedPorts
}

func generateBackendID(ingress *v1beta1.Ingress, rule *v1beta1.IngressRule, path *v1beta1.HTTPIngressPath, backend *v1beta1.IngressBackend) backendIdentifier {
	return backendIdentifier{
		serviceIdentifier: serviceIdentifier{
			Namespace: ingress.Namespace,
			Name:      backend.ServiceName,
		},
		Ingress: ingress,
		Rule:    rule,
		Path:    path,
		Backend: backend,
	}
}

func generateListenerID(rule *v1beta1.IngressRule,
	protocol network.ApplicationGatewayProtocol, overridePort *int32) listenerIdentifier {
	frontendPort := int32(80)
	if protocol == network.HTTPS {
		frontendPort = int32(443)
	}
	if overridePort != nil {
		frontendPort = *overridePort
	}
	listenerID := listenerIdentifier{
		FrontendPort: frontendPort,
		HostName:     rule.Host,
	}
	return listenerID
}

// GetApplicationGatewayPropertiesFormatPtr gets a pointer to updated ApplicationGatewayPropertiesFormat.
func (c *appGwConfigBuilder) GetApplicationGatewayPropertiesFormatPtr() *network.ApplicationGatewayPropertiesFormat {
	return &c.appGwConfig
}

type valFunc func(eventRecorder record.EventRecorder, config *network.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables, ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error

// PreBuildValidate runs all the validators that suggest misconfiguration in Kubernetes resources.
func (c *appGwConfigBuilder) PreBuildValidate(envVariables environment.EnvVariables, ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error {
	validationFunctions := []valFunc{
		validateServiceDefinition,
	}

	return c.runValidationFunctions(envVariables, ingressList, serviceList, validationFunctions)
}

// PostBuildValidate runs all the validators on the config constructed to ensure it complies with App Gateway requirements.
func (c *appGwConfigBuilder) PostBuildValidate(envVariables environment.EnvVariables, ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error {
	validationFunctions := []valFunc{
		validateURLPathMaps,
	}

	return c.runValidationFunctions(envVariables, ingressList, serviceList, validationFunctions)
}

func (c *appGwConfigBuilder) runValidationFunctions(envVariables environment.EnvVariables, ingressList []*v1beta1.Ingress, serviceList []*v1.Service, validationFunctions []valFunc) error {
	for _, fn := range validationFunctions {
		if err := fn(c.recorder, &c.appGwConfig, envVariables, ingressList, serviceList); err != nil {
			return err
		}
	}

	return nil
}
