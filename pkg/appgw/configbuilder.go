// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure/tags"
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

type memoization struct {
	listeners                    *[]n.ApplicationGatewayHTTPListener
	listenerConfigs              *map[listenerIdentifier]listenerAzConfig
	routingRules                 *[]n.ApplicationGatewayRequestRoutingRule
	pathMaps                     *[]n.ApplicationGatewayURLPathMap
	probesByName                 *map[string]n.ApplicationGatewayProbe
	probesByBackend              *map[backendIdentifier]*n.ApplicationGatewayProbe
	backendIDs                   *map[backendIdentifier]interface{}
	settings                     *[]n.ApplicationGatewayBackendHTTPSettings
	settingsByBackend            *map[backendIdentifier]*n.ApplicationGatewayBackendHTTPSettings
	serviceBackendPairsByBackend *map[backendIdentifier]serviceBackendPortPair
	pools                        *[]n.ApplicationGatewayBackendAddressPool
	certs                        *[]n.ApplicationGatewaySslCertificate
	redirectConfigs              *[]n.ApplicationGatewayRedirectConfiguration
	ports                        *[]n.ApplicationGatewayFrontendPort
}

type appGwConfigBuilder struct {
	k8sContext      *k8scontext.Context
	appGwIdentifier Identifier
	appGw           n.ApplicationGateway
	recorder        record.EventRecorder
	mem             memoization
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
		glog.Errorf("unable to generate Health Probes, error [%v]", err)
		return nil, ErrGeneratingProbes
	}

	err = c.BackendHTTPSettingsCollection(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate backend http settings, error [%v]", err)
		return nil, ErrGeneratingBackendSettings
	}

	// BackendAddressPools depend on BackendHTTPSettings
	err = c.BackendAddressPools(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate backend address pools, error [%v]", err)
		return nil, ErrCreatingBackendPools
	}

	// Listener configures the frontend listeners
	// This also creates redirection configuration (if TLS is configured and Ingress is annotated).
	// This configuration must be attached to request routing rules, which are created in the steps below.
	// The order of operations matters.
	err = c.Listeners(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate frontend listeners, error [%v]", err)
		return nil, ErrGeneratingListeners
	}

	// SSL redirection configurations created elsewhere will be attached to the appropriate rule in this step.
	err = c.RequestRoutingRules(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate request routing rules, error [%v]", err)
		return nil, ErrGeneratingRoutingRules
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

func generateListenerID(rule *v1beta1.IngressRule, protocol n.ApplicationGatewayProtocol, overridePort *Port, usePrivateIP bool) listenerIdentifier {
	frontendPort := Port(80)
	if protocol == n.HTTPS {
		frontendPort = Port(443)
	}
	if overridePort != nil {
		frontendPort = *overridePort
	}
	listenerID := listenerIdentifier{
		FrontendPort: frontendPort,
		HostName:     rule.Host,
		UsePrivateIP: usePrivateIP,
	}
	return listenerID
}

// addTags will add certain tags to Application Gateway
func (c *appGwConfigBuilder) addTags() {
	if c.appGw.Tags == nil {
		c.appGw.Tags = make(map[string]*string)
	}
	// Identify the App Gateway as being exclusively managed by a Kubernetes Ingress.
	c.appGw.Tags[tags.ManagedByK8sIngress] = to.StringPtr(fmt.Sprintf("%s/%s/%s", version.Version, version.GitCommit, version.BuildDate))
	if aksResourceID, err := azure.ConvertToClusterResourceGroup(c.k8sContext.GetInfrastructureResourceGroupID()); err == nil {
		c.appGw.Tags[tags.IngressForAKSClusterID] = to.StringPtr(aksResourceID)
	} else {
		glog.V(5).Infof("Error while parsing cluster resource ID for tagging: %s", err)
	}
}
