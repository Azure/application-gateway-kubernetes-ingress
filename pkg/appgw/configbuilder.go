// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/record"
)

// ConfigBuilder is a builder for application gateway configuration
type ConfigBuilder interface {
	// builder pattern
	BackendHTTPSettingsCollection(ingressList []*v1beta1.Ingress) error
	BackendAddressPools(ingressList []*v1beta1.Ingress) error
	Listeners(ingressList []*v1beta1.Ingress) error
	RequestRoutingRules(ingressList []*v1beta1.Ingress) error
	HealthProbesCollection(ingressList []*v1beta1.Ingress) error
	Build() *network.ApplicationGatewayPropertiesFormat
}

type appGwConfigBuilder struct {
	serviceBackendPairMap map[backendIdentifier](serviceBackendPortPair)

	backendHTTPSettingsMap map[backendIdentifier](*network.ApplicationGatewayBackendHTTPSettings)

	backendPoolMap map[backendIdentifier](*network.ApplicationGatewayBackendAddressPool)
	probesMap      map[backendIdentifier](*network.ApplicationGatewayProbe)

	k8sContext      *k8scontext.Context
	appGwIdentifier Identifier
	appGwConfig     network.ApplicationGatewayPropertiesFormat
	recorder        record.EventRecorder
}

// NewConfigBuilder construct a builder
func NewConfigBuilder(context *k8scontext.Context, appGwIdentifier *Identifier, originalConfig *network.ApplicationGatewayPropertiesFormat, recorder record.EventRecorder) ConfigBuilder {
	return &appGwConfigBuilder{
		// TODO(draychev): Decommission internal state
		serviceBackendPairMap:  make(map[backendIdentifier]serviceBackendPortPair),
		probesMap:              make(map[backendIdentifier]*network.ApplicationGatewayProbe),
		backendHTTPSettingsMap: make(map[backendIdentifier]*network.ApplicationGatewayBackendHTTPSettings),
		backendPoolMap:         make(map[backendIdentifier]*network.ApplicationGatewayBackendAddressPool),
		k8sContext:             context,
		appGwIdentifier:        *appGwIdentifier,
		appGwConfig:            *originalConfig,
		recorder:               recorder,
	}
}

// resolvePortName function goes through the endpoints of a given service and
// look for possible port number corresponding to a port name
func (c *appGwConfigBuilder) resolvePortName(portName string, backendID *backendIdentifier) map[int32]interface{} {
	endpoints := c.k8sContext.GetEndpointsByService(backendID.serviceKey())
	resolvedPorts := make(map[int32]interface{})
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

// Build generates the ApplicationGatewayPropertiesFormat for azure resource manager
func (c *appGwConfigBuilder) Build() *network.ApplicationGatewayPropertiesFormat {
	config := c.appGwConfig
	return &config
}
