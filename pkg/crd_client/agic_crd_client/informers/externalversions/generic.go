/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	v1beta1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewaybackendpool/v1beta1"
	azureapplicationgatewayinstanceupdatestatusv1beta1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayinstanceupdatestatus/v1beta1"
	v1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayloaddistributionpolicy/v1"
	azureingressprohibitedtargetv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=azureapplicationgatewaybackendpools.appgw.ingress.azure.io, Version=v1beta1
	case v1beta1.SchemeGroupVersion.WithResource("azureapplicationgatewaybackendpools"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Azureapplicationgatewaybackendpools().V1beta1().AzureApplicationGatewayBackendPools().Informer()}, nil

		// Group=azureapplicationgatewayinstanceupdatestatus.appgw.ingress.azure.io, Version=v1beta1
	case azureapplicationgatewayinstanceupdatestatusv1beta1.SchemeGroupVersion.WithResource("azureapplicationgatewayinstanceupdatestatuses"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Azureapplicationgatewayinstanceupdatestatus().V1beta1().AzureApplicationGatewayInstanceUpdateStatuses().Informer()}, nil

		// Group=azureapplicationgatewayloaddistributionpolicies.appgw.ingress.azure.io, Version=v1
	case v1.SchemeGroupVersion.WithResource("azureapplicationgatewayloaddistributionpolicies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Azureapplicationgatewayloaddistributionpolicy().V1().AzureApplicationGatewayLoadDistributionPolicies().Informer()}, nil

		// Group=azureingressprohibitedtargets.appgw.ingress.k8s.io, Version=v1
	case azureingressprohibitedtargetv1.SchemeGroupVersion.WithResource("azureingressprohibitedtargets"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Azureingressprohibitedtargets().V1().AzureIngressProhibitedTargets().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
