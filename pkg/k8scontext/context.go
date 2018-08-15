package k8scontext

import (
	"fmt"
	"reflect"
	"time"

	"github.com/eapache/channels"
	"github.com/golang/glog"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	informerv1 "k8s.io/client-go/informers/core/v1"
	informerv1beta1 "k8s.io/client-go/informers/extensions/v1beta1"
)

// InformerCollection : all the informers for k8s resources we care about.
type InformerCollection struct {
	Ingress   cache.SharedIndexInformer
	Endpoints cache.SharedIndexInformer
	Service   cache.SharedIndexInformer
}

// CacheCollection : all the listers from the informers.
type CacheCollection struct {
	Ingress           cache.Store
	IngressAnnotation cache.Store
	Service           cache.Store
	Endpoints         cache.Store
}

// Context : cache and listener for k8s resources.
type Context struct {
	informers   *InformerCollection
	Caches      *CacheCollection
	stopChannel chan struct{}

	UpdateChannel *channels.RingChannel
}

// NewContext creates a context based on a Kubernetes client instance.
func NewContext(kubeClient kubernetes.Interface, namespace string, resyncPeriod time.Duration) *Context {
	indexer := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}

	context := &Context{
		informers: &InformerCollection{
			Ingress:   informerv1beta1.NewIngressInformer(kubeClient, namespace, resyncPeriod, indexer),
			Service:   informerv1.NewServiceInformer(kubeClient, namespace, resyncPeriod, indexer),
			Endpoints: informerv1.NewEndpointsInformer(kubeClient, namespace, resyncPeriod, indexer),
		},
		Caches:        &CacheCollection{},
		stopChannel:   make(chan struct{}),
		UpdateChannel: channels.NewRingChannel(1024),
	}

	context.Caches.Ingress = context.informers.Ingress.GetStore()
	context.Caches.Service = context.informers.Service.GetStore()
	context.Caches.Endpoints = context.informers.Endpoints.GetStore()

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

	// Register event handlers.
	context.informers.Ingress.AddEventHandler(resourceHandler)
	context.informers.Endpoints.AddEventHandler(resourceHandler)
	context.informers.Service.AddEventHandler(resourceHandler)

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
		if hasHTTPRule {
			ingressList = append(ingressList, ingress)
		}
	}
	return ingressList
}

// GetEndpointsByService returns the endpoints associated with a specific service.
func (c *Context) GetEndpointsByService(serviceKey string) *v1.Endpoints {
	endpointsInterface, exist, err := c.Caches.Endpoints.GetByKey(serviceKey)

	if err != nil {
		glog.V(1).Infof("unable to get endpoints from store, error occured %s", err.Error())
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
		glog.V(1).Infof("unable to get service from store, error occured %s", err.Error())
		return nil
	}

	if !exist {
		glog.V(1).Infof("unable to get service from store, no such service %s", serviceKey)
		return nil
	}

	service := serviceInterface.(*v1.Service)
	return service
}

// Run function starts all the infomers and waits for an initial sync.
func (i *InformerCollection) Run(stopCh chan struct{}) {
	go i.Endpoints.Run(stopCh)
	go i.Service.Run(stopCh)

	glog.V(1).Infoln("start waiting for initial cache sync")
	if !cache.WaitForCacheSync(stopCh, i.Endpoints.HasSynced, i.Service.HasSynced) {
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
