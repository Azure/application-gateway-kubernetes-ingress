// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// ConfigBuilderContext holds the structs we have fetches from Kubernetes + environment, based on which
// we will construct App Gateway config.
type ConfigBuilderContext struct {
	IngressList          []*networking.Ingress
	ServiceList          []*v1.Service
	ProhibitedTargets    []*ptv1.AzureIngressProhibitedTarget
	EnvVariables         environment.EnvVariables
	IstioGateways        []*v1alpha3.Gateway
	IstioVirtualServices []*v1alpha3.VirtualService

	DefaultAddressPoolID            *string
	DefaultLoadDistributionPolicyID *string
	DefaultHTTPSettingsID           *string

	ExistingPortsByNumber map[Port]n.ApplicationGatewayFrontendPort
}

// InIngressList returns true if an ingress is in the ingress list
func (cbCtx *ConfigBuilderContext) InIngressList(ingress *networking.Ingress) bool {
	for _, prunedIngress := range cbCtx.IngressList {
		if ingress.Name == prunedIngress.Name && ingress.Namespace == prunedIngress.Namespace {
			return true
		}
	}
	return false
}

// Port is a type alias for int32, representing a port number.
type Port int32
