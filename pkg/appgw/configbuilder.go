// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"errors"
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

// ConfigBuilder is a builder for application gateway configuration
type ConfigBuilder interface {
	PreBuildValidate(cbCtx *ConfigBuilderContext) error
	Build(cbCtx *ConfigBuilderContext) (*n.ApplicationGateway, error)
	PostBuildValidate(cbCtx *ConfigBuilderContext) error
}

type appGwConfigBuilder struct {
	k8sContext      *k8scontext.Context
	appGwIdentifier Identifier
	appGw           n.ApplicationGateway
	recorder        record.EventRecorder
}

// NewConfigBuilder construct a builder
func NewConfigBuilder(context *k8scontext.Context, appGwIdentifier *Identifier, original *n.ApplicationGateway, recorder record.EventRecorder) ConfigBuilder {
	return &appGwConfigBuilder{
		k8sContext:      context,
		appGwIdentifier: *appGwIdentifier,
		appGw:           *original,
		recorder:        recorder,
	}
}

// Build gets a pointer to updated ApplicationGatewayPropertiesFormat.
func (c *appGwConfigBuilder) Build(cbCtx *ConfigBuilderContext) (*n.ApplicationGateway, error) {
	err := c.HealthProbesCollection(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate Health Probes, error [%v]", err.Error())
		return nil, errors.New("unable to generate health probes")
	}

	err = c.BackendHTTPSettingsCollection(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate backend http settings, error [%v]", err.Error())
		return nil, errors.New("unable to generate backend http settings")
	}

	// BackendAddressPools depend on BackendHTTPSettings
	err = c.BackendAddressPools(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate backend address pools, error [%v]", err.Error())
		return nil, errors.New("unable to generate backend address pools")
	}

	// HTTPListener configures the frontend listeners
	// This also creates redirection configuration (if TLS is configured and Ingress is annotated).
	// This configuration must be attached to request routing rules, which are created in the steps below.
	// The order of operations matters.
	err = c.Listeners(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate frontend listeners, error [%v]", err.Error())
		return nil, errors.New("unable to generate frontend listeners")
	}

	// SSL redirection configurations created elsewhere will be attached to the appropriate rule in this step.
	err = c.RequestRoutingRules(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate request routing rules, error [%v]", err.Error())
		return nil, errors.New("unable to generate request routing rules")
	}

	c.addTags()

	return &c.appGw, nil
}

type valFunc func(eventRecorder record.EventRecorder, config *n.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables, ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error

// PreBuildValidate runs all the validators that suggest misconfiguration in Kubernetes resources.
func (c *appGwConfigBuilder) PreBuildValidate(cbCtx *ConfigBuilderContext) error {

	validationFunctions := []valFunc{
		validateServiceDefinition,
	}

	return c.runValidationFunctions(cbCtx, validationFunctions)
}

// PostBuildValidate runs all the validators on the config constructed to ensure it complies with App Gateway requirements.
func (c *appGwConfigBuilder) PostBuildValidate(cbCtx *ConfigBuilderContext) error {
	validationFunctions := []valFunc{
		validateURLPathMaps,
	}

	return c.runValidationFunctions(cbCtx, validationFunctions)
}

func (c *appGwConfigBuilder) runValidationFunctions(cbCtx *ConfigBuilderContext, validationFunctions []valFunc) error {
	for _, fn := range validationFunctions {
		if err := fn(c.recorder, c.appGw.ApplicationGatewayPropertiesFormat, cbCtx.EnvVariables, cbCtx.IngressList, cbCtx.ServiceList); err != nil {
			return err
		}
	}

	return nil
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

func generateIstioBackendID(virtualService *v1alpha3.VirtualService, rule *v1alpha3.HTTPRoute, match *v1alpha3.HTTPMatchRequest, destination *v1alpha3.Destination) istioBackendIdentifier {
	return istioBackendIdentifier{
		serviceIdentifier: serviceIdentifier{
			Namespace:	virtualService.Namespace,
			Name:		destination.Name,
		},
		VirtualService: virtualService,
		Rule: rule,
		Match: match,
		Destination: destination,
		Gateways: match.Gateways,
	}
}

func generateListenerID(rule *v1beta1.IngressRule,
	protocol n.ApplicationGatewayProtocol, overridePort *int32) listenerIdentifier {
	frontendPort := int32(80)
	if protocol == n.HTTPS {
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

// addTags will add certain tags to Application Gateway
func (c *appGwConfigBuilder) addTags() {
	if c.appGw.Tags == nil {
		c.appGw.Tags = make(map[string]*string)
	}
	// Identify the App Gateway as being exclusively managed by a Kubernetes Ingress.
	c.appGw.Tags[managedByK8sIngress] = to.StringPtr(fmt.Sprintf("%s/%s/%s", version.Version, version.GitCommit, version.BuildDate))
}

func (c *appGwConfigBuilder) getExistingBackendPools() []n.ApplicationGatewayBackendAddressPool {
	if c.appGw.BackendAddressPools == nil {
		return []n.ApplicationGatewayBackendAddressPool{}
	}
	return *c.appGw.BackendAddressPools
}

func (c *appGwConfigBuilder) getExistingListeners() []n.ApplicationGatewayHTTPListener {
	if c.appGw.HTTPListeners == nil {
		return []n.ApplicationGatewayHTTPListener{}
	}
	return *c.appGw.HTTPListeners
}

func (c *appGwConfigBuilder) getExistingRoutingRules() []n.ApplicationGatewayRequestRoutingRule {
	if c.appGw.RequestRoutingRules == nil {
		return []n.ApplicationGatewayRequestRoutingRule{}
	}
	return *c.appGw.RequestRoutingRules
}

func (c *appGwConfigBuilder) getExistingPathMaps() []n.ApplicationGatewayURLPathMap {
	if c.appGw.URLPathMaps == nil {
		return []n.ApplicationGatewayURLPathMap{}
	}
	return *c.appGw.URLPathMaps
}
