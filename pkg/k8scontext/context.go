// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"fmt"
	"reflect"
	"time"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/eapache/channels"
	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// InformerCollection : all the informers for k8s resources we care about.
type InformerCollection struct {
	Ingress   cache.SharedIndexInformer
	Endpoints cache.SharedIndexInformer
	Service   cache.SharedIndexInformer
	Secret    cache.SharedIndexInformer
}

// CacheCollection : all the listers from the informers.
type CacheCollection struct {
	Ingress   cache.Store
	Service   cache.Store
	Endpoints cache.Store
	Secret    cache.Store
}

// Context : cache and listener for k8s resources.
type Context struct {
	informers              *InformerCollection
	Caches                 *CacheCollection
	CertificateSecretStore SecretStore

	ingressSecretsMap utils.ThreadsafeMultiMap
	stopChannel       chan struct{}

	UpdateChannel *channels.RingChannel
}

// NewContext creates a context based on a Kubernetes client instance.
func NewContext(kubeClient kubernetes.Interface, namespace string, resyncPeriod time.Duration) *Context {
	informerFactory := informers.NewFilteredSharedInformerFactory(kubeClient, resyncPeriod, namespace, func(*metav1.ListOptions) {})

	context := &Context{
		informers: &InformerCollection{
			Ingress:   informerFactory.Extensions().V1beta1().Ingresses().Informer(),
			Service:   informerFactory.Core().V1().Services().Informer(),
			Endpoints: informerFactory.Core().V1().Endpoints().Informer(),
			Secret:    informerFactory.Core().V1().Secrets().Informer(),
		},
		ingressSecretsMap:      utils.NewThreadsafeMultimap(),
		Caches:                 &CacheCollection{},
		CertificateSecretStore: NewSecretStore(),
		stopChannel:            make(chan struct{}),
		UpdateChannel:          channels.NewRingChannel(1024),
	}

	context.Caches.Ingress = context.informers.Ingress.GetStore()
	context.Caches.Service = context.informers.Service.GetStore()
	context.Caches.Endpoints = context.informers.Endpoints.GetStore()
	context.Caches.Secret = context.informers.Secret.GetStore()

	addFunc := func(obj interface{}) {
		context.UpdateChannel.In() <- Event{
			Type:  Create,
			Value: obj,
		}
	}

	updateFunc := func(oldObj, newObj interface{}) {
		if reflect.DeepEqual(oldObj, newObj) {
			return
		}
		context.UpdateChannel.In() <- Event{
			Type:  Update,
			Value: newObj,
		}
	}

	deleteFunc := func(obj interface{}) {
		context.UpdateChannel.In() <- Event{
			Type:  Delete,
			Value: obj,
		}
	}

	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    addFunc,
		UpdateFunc: updateFunc,
		DeleteFunc: deleteFunc,
	}

	// ingress resource handlers
	ingressAddFunc := func(obj interface{}) {
		ing := obj.(*v1beta1.Ingress)

		if !isIngressApplicationGateway(ing) {
			return
		}

		if ing.Spec.TLS != nil && len(ing.Spec.TLS) > 0 {
			ingKey := utils.GetResourceKey(ing.Namespace, ing.Name)
			for _, tls := range ing.Spec.TLS {
				secKey := utils.GetResourceKey(ing.Namespace, tls.SecretName)

				if context.ingressSecretsMap.ContainsPair(ingKey, secKey) {
					continue
				}

				if secret, exists, err := context.Caches.Secret.GetByKey(secKey); exists && err == nil {
					if !context.ingressSecretsMap.ContainsValue(secKey) {
						done := context.CertificateSecretStore.convertSecret(secKey, secret.(*v1.Secret))
						if !done {
							continue
						}
					}
				}

				context.ingressSecretsMap.Insert(ingKey, secKey)
			}
		}
		context.UpdateChannel.In() <- Event{
			Type:  Create,
			Value: obj,
		}
	}

	ingressUpdateFunc := func(oldObj, newObj interface{}) {
		if reflect.DeepEqual(oldObj, newObj) {
			return
		}
		oldIng := oldObj.(*v1beta1.Ingress)
		ing := newObj.(*v1beta1.Ingress)
		if !isIngressApplicationGateway(ing) && !isIngressApplicationGateway(oldIng) {
			return
		}
		if ing.Spec.TLS != nil && len(ing.Spec.TLS) > 0 {
			ingKey := utils.GetResourceKey(ing.Namespace, ing.Name)
			context.ingressSecretsMap.Clear(ingKey)
			for _, tls := range ing.Spec.TLS {
				secKey := utils.GetResourceKey(ing.Namespace, tls.SecretName)

				if context.ingressSecretsMap.ContainsPair(ingKey, secKey) {
					continue
				}

				if secret, exists, err := context.Caches.Secret.GetByKey(secKey); exists && err == nil {
					if !context.ingressSecretsMap.ContainsValue(secKey) {
						done := context.CertificateSecretStore.convertSecret(secKey, secret.(*v1.Secret))
						if !done {
							continue
						}
					}
				}

				context.ingressSecretsMap.Insert(ingKey, secKey)
			}
		}

		context.UpdateChannel.In() <- Event{
			Type:  Update,
			Value: newObj,
		}
	}

	ingressDeleteFunc := func(obj interface{}) {
		ing, ok := obj.(*v1beta1.Ingress)
		if !ok {
			tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
			if !ok {
				// unable to get from tombstone
				return
			}
			ing, ok = tombstone.Obj.(*v1beta1.Ingress)
		}
		if ing == nil {
			return
		}
		if !isIngressApplicationGateway(ing) {
			return
		}
		ingKey := utils.GetResourceKey(ing.Namespace, ing.Name)
		context.ingressSecretsMap.Erase(ingKey)

		context.UpdateChannel.In() <- Event{
			Type:  Delete,
			Value: obj,
		}
	}

	ingressResourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    ingressAddFunc,
		UpdateFunc: ingressUpdateFunc,
		DeleteFunc: ingressDeleteFunc,
	}

	// secret resource handlers
	secretAddFunc := func(obj interface{}) {
		sec := obj.(*v1.Secret)
		secKey := utils.GetResourceKey(sec.Namespace, sec.Name)
		if context.ingressSecretsMap.ContainsValue(secKey) {
			// find if this secKey exists in the map[string]UnorderedSets
			done := context.CertificateSecretStore.convertSecret(secKey, sec)
			if done {
				context.UpdateChannel.In() <- Event{
					Type:  Create,
					Value: obj,
				}
			}
		}
	}

	secretUpdateFunc := func(oldObj, newObj interface{}) {
		if reflect.DeepEqual(oldObj, newObj) {
			return
		}

		sec := newObj.(*v1.Secret)
		secKey := utils.GetResourceKey(sec.Namespace, sec.Name)
		if context.ingressSecretsMap.ContainsValue(secKey) {
			done := context.CertificateSecretStore.convertSecret(secKey, sec)
			if done {
				context.UpdateChannel.In() <- Event{
					Type:  Update,
					Value: newObj,
				}
			}
		}
	}

	secretDeleteFunc := func(obj interface{}) {
		sec, ok := obj.(*v1.Secret)
		if !ok {
			tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
			if !ok {
				// unable to get from tombstone
				return
			}
			sec, ok = tombstone.Obj.(*v1.Secret)
		}
		if sec == nil {
			return
		}

		secKey := utils.GetResourceKey(sec.Namespace, sec.Name)
		context.CertificateSecretStore.eraseSecret(secKey)
		if context.ingressSecretsMap.ContainsValue(secKey) {
			context.UpdateChannel.In() <- Event{
				Type:  Delete,
				Value: obj,
			}
		}
	}

	secretResourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    secretAddFunc,
		UpdateFunc: secretUpdateFunc,
		DeleteFunc: secretDeleteFunc,
	}

	// Register event handlers.
	context.informers.Endpoints.AddEventHandler(resourceHandler)
	context.informers.Service.AddEventHandler(resourceHandler)
	context.informers.Secret.AddEventHandler(secretResourceHandler)
	context.informers.Ingress.AddEventHandler(ingressResourceHandler)

	return context
}

// Run executes informer collection.
func (c *Context) Run() {
	glog.V(1).Infoln("k8s context run started")
	c.informers.Run(c.stopChannel)
	glog.V(1).Infoln("k8s context run finished")
}

// GetHTTPIngressList returns a list of all the ingresses for HTTP from cache.
func (c *Context) GetHTTPIngressList() []*v1beta1.Ingress {
	ingressListInterface := c.Caches.Ingress.List()
	ingressList := make([]*v1beta1.Ingress, 0)
	for _, ingressInterface := range ingressListInterface {
		ingress := ingressInterface.(*v1beta1.Ingress)

		hasHTTPRule := false
		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP != nil {
				hasHTTPRule = true
				break
			}
		}

		if hasHTTPRule && isIngressApplicationGateway(ingress) {
			ingressList = append(ingressList, ingress)
		}
	}
	return ingressList
}

// GetEndpointsByService returns the endpoints associated with a specific service.
func (c *Context) GetEndpointsByService(serviceKey string) *v1.Endpoints {
	endpointsInterface, exist, err := c.Caches.Endpoints.GetByKey(serviceKey)

	if err != nil {
		glog.V(1).Infof("unable to get endpoints from store, error occurred %s", err.Error())
		return nil
	}

	if !exist {
		glog.V(1).Infof("unable to get endpoints from store, no such service %s", serviceKey)
		return nil
	}

	endpoints := endpointsInterface.(*v1.Endpoints)
	return endpoints
}

// GetService returns the service identified by the key.
func (c *Context) GetService(serviceKey string) *v1.Service {
	serviceInterface, exist, err := c.Caches.Service.GetByKey(serviceKey)

	if err != nil {
		glog.V(1).Infof("unable to get service from store, error occurred %s", err.Error())
		return nil
	}

	if !exist {
		glog.V(1).Infof("unable to get service from store, no such service %s", serviceKey)
		return nil
	}

	service := serviceInterface.(*v1.Service)
	return service
}

// GetSecret returns the secret identified by the key
func (c *Context) GetSecret(secretKey string) *v1.Secret {
	secretInterface, exist, err := c.Caches.Secret.GetByKey(secretKey)

	if err != nil {
		glog.V(1).Infof("unable to get secret from store, error occurred %s", err.Error())
		return nil
	}

	if !exist {
		glog.V(1).Infof("unable to get secret from store, no such service %s", secretKey)
		return nil
	}

	secret := secretInterface.(*v1.Secret)
	return secret
}

// Run function starts all the informers and waits for an initial sync.
func (i *InformerCollection) Run(stopCh chan struct{}) {
	go i.Endpoints.Run(stopCh)
	go i.Service.Run(stopCh)
	go i.Secret.Run(stopCh)

	glog.V(1).Infoln("start waiting for initial cache sync")
	if !cache.WaitForCacheSync(stopCh, i.Endpoints.HasSynced, i.Service.HasSynced, i.Secret.HasSynced) {
		glog.V(1).Infoln("initial sync wait stopped")
		runtime.HandleError(fmt.Errorf("failed to do initial sync on resources required for ingress"))
		return
	}

	go i.Ingress.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, i.Ingress.HasSynced) {
		glog.V(1).Infoln("ingress cache wait stopped")
		runtime.HandleError(fmt.Errorf("failed to do initial sync on ingress"))
		return
	}

	glog.V(1).Infoln("ingress initial sync done")
}

// Stop function stops all informers in the context.
func (c *Context) Stop() {
	c.stopChannel <- struct{}{}
}

func isIngressApplicationGateway(ingress *v1beta1.Ingress) bool {
	controllerName := ingress.Annotations["kubernetes.io/ingress.class"]
	return controllerName == "azure/application-gateway"
}
