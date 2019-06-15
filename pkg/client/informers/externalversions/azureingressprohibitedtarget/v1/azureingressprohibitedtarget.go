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
	time "time"

	azureingressprohibitedtargetv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/client/clientset/versioned"
	internalinterfaces "github.com/Azure/application-gateway-kubernetes-ingress/pkg/client/informers/externalversions/internalinterfaces"
	v1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/client/listers/azureingressprohibitedtarget/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// AzureIngressProhibitedTargetInformer provides access to a shared informer and lister for
// AzureIngressProhibitedTargets.
type AzureIngressProhibitedTargetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.AzureIngressProhibitedTargetLister
}

type azureIngressProhibitedTargetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewAzureIngressProhibitedTargetInformer constructs a new informer for AzureIngressProhibitedTarget type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewAzureIngressProhibitedTargetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredAzureIngressProhibitedTargetInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredAzureIngressProhibitedTargetInformer constructs a new informer for AzureIngressProhibitedTarget type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredAzureIngressProhibitedTargetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AzureingressprohibitedtargetsV1().AzureIngressProhibitedTargets(namespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AzureingressprohibitedtargetsV1().AzureIngressProhibitedTargets(namespace).Watch(options)
			},
		},
		&azureingressprohibitedtargetv1.AzureIngressProhibitedTarget{},
		resyncPeriod,
		indexers,
	)
}

func (f *azureIngressProhibitedTargetInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredAzureIngressProhibitedTargetInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *azureIngressProhibitedTargetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&azureingressprohibitedtargetv1.AzureIngressProhibitedTarget{}, f.defaultInformer)
}

func (f *azureIngressProhibitedTargetInformer) Lister() v1.AzureIngressProhibitedTargetLister {
	return v1.NewAzureIngressProhibitedTargetLister(f.Informer().GetIndexer())
}
