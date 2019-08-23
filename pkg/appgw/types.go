// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// ConfigBuilderContext holds the structs we have fetches from Kubernetes + environment, based on which
// we will construct App Gateway config.
type ConfigBuilderContext struct {
	IngressList          []*v1beta1.Ingress
	ServiceList          []*v1.Service
	ProhibitedTargets    []*ptv1.AzureIngressProhibitedTarget
	EnvVariables         environment.EnvVariables
	IstioGateways        []*v1alpha3.Gateway
	IstioVirtualServices []*v1alpha3.VirtualService

	// Feature flag toggling Brownfield Deployment across the entire AGIC code base.
	EnableBrownfieldDeployment bool

	// Feature flag toggling Istio Integration across the entire AGIC code base.
	EnableIstioIntegration bool

	// Feature flag enabling panic() when put to ARM fails.
	EnablePanicOnPutError bool
}

// InIngressList returns true if an ingress is in the ingress list
func (cbCtx *ConfigBuilderContext) InIngressList(ingress *v1beta1.Ingress) bool {
	for _, prunedIngress := range cbCtx.IngressList {
		if ingress.Name == prunedIngress.Name && ingress.Namespace == prunedIngress.Namespace {
			return true
		}
	}
	return false
}

type Port int32
