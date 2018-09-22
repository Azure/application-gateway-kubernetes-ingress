// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"

	"k8s.io/api/extensions/v1beta1"
)

// ConfigBuilder is a builder for application gateway configuration
type ConfigBuilder interface {
	// builder pattern
	BackendHTTPSettingsCollection(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error)
	BackendAddressPools(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error)
	HTTPListeners(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error)
	RequestRoutingRules(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error)

	Build() *network.ApplicationGatewayPropertiesFormat
}

type appGwConfigBuilder struct {
	serviceBackendPairMap map[backendIdentifier](serviceBackendPortPair)

	ingressKeyHostnameSecretIDMap map[string](map[string]secretIdentifier)
	secretIDCertificateMap        map[secretIdentifier]*string

	httpListenersMap            map[frontendListenerIdentifier](*network.ApplicationGatewayHTTPListener)
	httpListenersAzureConfigMap map[frontendListenerIdentifier](*frontendListenerAzureConfig)

	backendHTTPSettingsMap map[backendIdentifier](*network.ApplicationGatewayBackendHTTPSettings)

	backendPoolMap map[backendIdentifier](*network.ApplicationGatewayBackendAddressPool)

	k8sContext      *k8scontext.Context
	appGwIdentifier Identifier
	appGwConfig     network.ApplicationGatewayPropertiesFormat
}

// NewConfigBuilder construct a builder
func NewConfigBuilder(context *k8scontext.Context, appGwIdentifier *Identifier, originalConfig *network.ApplicationGatewayPropertiesFormat) ConfigBuilder {
	return &appGwConfigBuilder{
		serviceBackendPairMap:         make(map[backendIdentifier](serviceBackendPortPair)),
		httpListenersMap:              make(map[frontendListenerIdentifier](*network.ApplicationGatewayHTTPListener)),
		httpListenersAzureConfigMap:   make(map[frontendListenerIdentifier](*frontendListenerAzureConfig)),
		ingressKeyHostnameSecretIDMap: make(map[string](map[string]secretIdentifier)),
		secretIDCertificateMap:        make(map[secretIdentifier]*string),
		backendHTTPSettingsMap:        make(map[backendIdentifier](*network.ApplicationGatewayBackendHTTPSettings)),
		backendPoolMap:                make(map[backendIdentifier](*network.ApplicationGatewayBackendAddressPool)),
		k8sContext:                    context,
		appGwIdentifier:               *appGwIdentifier,
		appGwConfig:                   *originalConfig,
	}
}

// resolvePortName function goes through the endpoints of a given service and
// look for possible port number corresponding to a port name
func (builder *appGwConfigBuilder) resolvePortName(portName string, backendID *backendIdentifier) utils.UnorderedSet {
	endpoints := builder.k8sContext.GetEndpointsByService(backendID.serviceKey())
	resolvedPorts := utils.NewUnorderedSet()
	for _, subset := range endpoints.Subsets {
		for _, epPort := range subset.Ports {
			if epPort.Name == portName {
				resolvedPorts.Insert(epPort.Port)
			}
		}
	}
	return resolvedPorts
}

func generateBackendID(ingress *v1beta1.Ingress, backend *v1beta1.IngressBackend) backendIdentifier {
	backendServiceName := backend.ServiceName
	backendServicePort := backend.ServicePort
	backendID := backendIdentifier{
		serviceIdentifier: serviceIdentifier{
			Namespace: ingress.Namespace,
			Name:      backendServiceName,
		},
		ServicePort: backendServicePort,
	}
	return backendID
}

func generateFrontendListenerID(rule *v1beta1.IngressRule,
	protocol network.ApplicationGatewayProtocol, overridePort *int32) frontendListenerIdentifier {
	frontendPort := int32(80)
	if protocol == network.HTTPS {
		frontendPort = int32(443)
	}
	if overridePort != nil {
		frontendPort = *overridePort
	}
	frontendListenerID := frontendListenerIdentifier{
		FrontendPort: frontendPort,
		HostName:     rule.Host,
	}
	return frontendListenerID
}

// Build generates the ApplicationGatewayPropertiesFormat for azure resource manager
func (builder *appGwConfigBuilder) Build() *network.ApplicationGatewayPropertiesFormat {
	config := builder.appGwConfig
	return &config
}
