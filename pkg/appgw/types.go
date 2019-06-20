package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

// ConfigBuilderContext holds the structs we have fetches from Kubernetes + environment, based on which
// we will construct App Gateway config.
type ConfigBuilderContext struct {
	IngressList       []*v1beta1.Ingress
	ServiceList       []*v1.Service
	ManagedTargets    []*mtv1.AzureIngressManagedTarget
	ProhibitedTargets []*ptv1.AzureIngressProhibitedTarget
	EnvVariables      environment.EnvVariables
}
