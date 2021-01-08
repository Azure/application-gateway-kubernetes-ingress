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

package v1

import (
	"context"
	time "time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"

	azureingressallowedtargetv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressallowedtarget/v1"
	versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	internalinterfaces "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/informers/externalversions/internalinterfaces"
	v1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/listers/azureingressallowedtarget/v1"
)

// AzureIngressAllowedTargetInformer provides access to a shared informer and lister for
// AzureIngressAllowedTargets.
type AzureIngressAllowedTargetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.AzureIngressAllowedTargetLister
}

type azureIngressAllowedTargetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewAzureIngressAllowedTargetInformer constructs a new informer for AzureIngressAllowedTarget type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewAzureIngressAllowedTargetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredAzureIngressAllowedTargetInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredAzureIngressAllowedTargetInformer constructs a new informer for AzureIngressAllowedTarget type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredAzureIngressAllowedTargetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AzureingressallowedtargetsV1().AzureIngressAllowedTargets(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AzureingressallowedtargetsV1().AzureIngressAllowedTargets(namespace).Watch(context.TODO(), options)
			},
		},
		&azureingressallowedtargetv1.AzureIngressAllowedTarget{},
		resyncPeriod,
		indexers,
	)
}

func (f *azureIngressAllowedTargetInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredAzureIngressAllowedTargetInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *azureIngressAllowedTargetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&azureingressallowedtargetv1.AzureIngressAllowedTarget{}, f.defaultInformer)
}

func (f *azureIngressAllowedTargetInformer) Lister() v1.AzureIngressAllowedTargetLister {
	return v1.NewAzureIngressAllowedTargetLister(f.Informer().GetIndexer())
}