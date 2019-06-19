// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"errors"
	"fmt"
	"time"

	"github.com/deckarep/golang-set"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	managedv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	prohibitedv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/client/clientset/versioned"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/client/informers/externalversions"
	istio_versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/istio_client/clientset/versioned"
	istio_externalversions "github.com/Azure/application-gateway-kubernetes-ingress/pkg/istio_client/informers/externalversions"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// NewContext creates a context based on a Kubernetes client instance.
func NewContext(istioCrdClient istio_versioned.Interface, kubeClient kubernetes.Interface, crdClient versioned.Interface, namespaces []string, resyncPeriod time.Duration) *Context {
	updateChannel := channels.NewRingChannel(1024)

	var options []informers.SharedInformerOption
	var crdOptions []externalversions.SharedInformerOption
	for _, namespace := range namespaces {
		options = append(options, informers.WithNamespace(namespace))
		crdOptions = append(crdOptions, externalversions.WithNamespace(namespace))
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, options...)
	istioGwy := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod)
	crdInformerFactory := externalversions.NewSharedInformerFactoryWithOptions(crdClient, resyncPeriod, crdOptions...)
	istioGwy := istio_externalversions.NewSharedInformerFactoryWithOptions(istioCrdClient, resyncPeriod)

	informerCollection := InformerCollection{
		Endpoints:    informerFactory.Core().V1().Endpoints().Informer(),
		Ingress:      informerFactory.Extensions().V1beta1().Ingresses().Informer(),
		Pods:         informerFactory.Core().V1().Pods().Informer(),
		Secret:       informerFactory.Core().V1().Secrets().Informer(),
		Service:      informerFactory.Core().V1().Services().Informer(),
		IstioGateway: istioGwy.Networking().V1alpha3().Gateways().Informer(),

		AzureIngressManagedLocation:    crdInformerFactory.Azureingressmanagedtargets().V1().AzureIngressManagedTargets().Informer(),
		AzureIngressProhibitedLocation: crdInformerFactory.Azureingressprohibitedtargets().V1().AzureIngressProhibitedTargets().Informer(),
	}

	cacheCollection := CacheCollection{
		Endpoints:                      informerCollection.Endpoints.GetStore(),
		Ingress:                        informerCollection.Ingress.GetStore(),
		Pods:                           informerCollection.Pods.GetStore(),
		Secret:                         informerCollection.Secret.GetStore(),
		Service:                        informerCollection.Service.GetStore(),
		AzureIngressManagedLocation:    informerCollection.AzureIngressManagedLocation.GetStore(),
		AzureIngressProhibitedLocation: informerCollection.AzureIngressProhibitedLocation.GetStore(),
	}

	context := &Context{
		informers:              &informerCollection,
		ingressSecretsMap:      utils.NewThreadsafeMultimap(),
		Caches:                 &cacheCollection,
		CertificateSecretStore: NewSecretStore(),
		stopChannel:            make(chan struct{}),
		UpdateChannel:          updateChannel,
	}

	h := handlers{context}

	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.addFunc,
		UpdateFunc: h.updateFunc,
		DeleteFunc: h.deleteFunc,
	}

	ingressResourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.ingressAddFunc,
		UpdateFunc: h.ingressUpdateFunc,
		DeleteFunc: h.ingressDeleteFunc,
	}

	secretResourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.secretAddFunc,
		UpdateFunc: h.secretUpdateFunc,
		DeleteFunc: h.secretDeleteFunc,
	}

	// Register event handlers.
	informerCollection.Endpoints.AddEventHandler(resourceHandler)
	informerCollection.Ingress.AddEventHandler(ingressResourceHandler)
	informerCollection.Pods.AddEventHandler(resourceHandler)
	informerCollection.Secret.AddEventHandler(secretResourceHandler)
	informerCollection.Service.AddEventHandler(resourceHandler)

	return context
}

// Run executes informer collection.
func (c *Context) Run(omitCRDs bool) {
	glog.V(1).Infoln("k8s context run started")
	c.informers.Run(c.stopChannel, omitCRDs)
	glog.V(1).Infoln("k8s context run finished")
}

// GetServiceList returns a list of all the Services from cache.
func (c *Context) GetServiceList() []*v1.Service {
	var serviceList []*v1.Service
	for _, ingressInterface := range c.Caches.Service.List() {
		service := ingressInterface.(*v1.Service)
		if hasTCPPort(service) {
			serviceList = append(serviceList, service)
		}
	}
	return serviceList
}

func hasTCPPort(service *v1.Service) bool {
	for _, port := range service.Spec.Ports {
		if port.Protocol == v1.ProtocolTCP {
			return true
		}
	}
	return false
}

// GetHTTPIngressList returns a list of all the ingresses for HTTP from cache.
func (c *Context) GetHTTPIngressList() []*v1beta1.Ingress {
	var ingressList []*v1beta1.Ingress
	for _, ingressInterface := range c.Caches.Ingress.List() {
		ingress := ingressInterface.(*v1beta1.Ingress)
		if hasHTTPRule(ingress) && isIngressApplicationGateway(ingress) {
			ingressList = append(ingressList, ingress)
		}
	}
	return ingressList
}

// GetAzureIngressManagedTargets returns a list of App Gwy configs, for which AGIC is explicitly allowed to modify config.
func (c *Context) GetAzureIngressManagedTargets() []*managedv1.AzureIngressManagedTarget {
	var targets []*managedv1.AzureIngressManagedTarget
	for _, obj := range c.Caches.AzureIngressManagedLocation.List() {
		targets = append(targets, obj.(*managedv1.AzureIngressManagedTarget))
	}
	return targets
}

// GetAzureProhibitedTargets returns a list of App Gwy configs, for which AGIC is not allowed to modify config.
func (c *Context) GetAzureProhibitedTargets() []*prohibitedv1.AzureIngressProhibitedTarget {
	var targets []*prohibitedv1.AzureIngressProhibitedTarget
	for _, obj := range c.Caches.AzureIngressProhibitedLocation.List() {
		targets = append(targets, obj.(*prohibitedv1.AzureIngressProhibitedTarget))
	}
	return targets
}

func hasHTTPRule(ingress *v1beta1.Ingress) bool {
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP != nil {
			return true
		}
	}
	return false
}

// GetEndpointsByService returns the endpoints associated with a specific service.
func (c *Context) GetEndpointsByService(serviceKey string) (*v1.Endpoints, error) {
	endpointsInterface, exist, err := c.Caches.Endpoints.GetByKey(serviceKey)

	if err != nil {
		glog.Error("Error fetching endpoints from store, error occurred ", err)
		return nil, err
	}

	if !exist {
		msg := fmt.Sprintf("Error fetching endpoints from store! Service does not exist: %s", serviceKey)
		glog.Error(msg)
		return nil, errors.New(msg)
	}

	return endpointsInterface.(*v1.Endpoints), nil
}

// GetService returns the service identified by the key.
// Deprecated: Please use a map instead
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

// GetPodsByServiceSelector returns pods that are associated with a specific service.
func (c *Context) GetPodsByServiceSelector(selector map[string]string) []*v1.Pod {
	selectorSet := mapset.NewSet()
	for k, v := range selector {
		selectorSet.Add(k + ":" + v)
	}

	var podList []*v1.Pod
	for _, podInterface := range c.Caches.Pods.List() {
		pod := podInterface.(*v1.Pod)
		podLabelSet := mapset.NewSet()
		for k, v := range pod.Labels {
			podLabelSet.Add(k + ":" + v)
		}

		if selectorSet.IsSubset(podLabelSet) {
			podList = append(podList, pod)
		}
	}

	return podList
}

// GetSecret returns the secret identified by the key
func (c *Context) GetSecret(secretKey string) *v1.Secret {
	secretInterface, exist, err := c.Caches.Secret.GetByKey(secretKey)

	if err != nil {
		glog.Error("Error fetching secret from store:", err)
		return nil
	}

	if !exist {
		glog.Error("Error fetching secret from store! Service does not exist:", secretKey)
		return nil
	}

	secret := secretInterface.(*v1.Secret)
	return secret
}

// Run function starts all the informers and waits for an initial sync.
func (i *InformerCollection) Run(stopCh chan struct{}, omitCRDs bool) {
	var hasSynced []cache.InformerSynced
	crds := map[cache.SharedInformer]interface{}{
		i.AzureIngressManagedLocation:    nil,
		i.AzureIngressProhibitedLocation: nil,
	}
	sharedInformers := []cache.SharedInformer{
		i.Endpoints,
		i.Pods,
		i.Service,
		i.Secret,
		i.Ingress,
		i.AzureIngressManagedLocation,
		i.AzureIngressProhibitedLocation,
	}

	for _, informer := range sharedInformers {
		go informer.Run(stopCh)
		// NOTE: Delyan could not figure out how to make informer.HasSynced == true for the CRDs in unit tests
		// so until we do that - we omit WaitForCacheSync for CRDs in unit testing
		if _, isCRD := crds[informer]; isCRD {
			continue
		}
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	glog.V(1).Infoln("Wait for initial cache sync")
	if !cache.WaitForCacheSync(stopCh, hasSynced...) {
		glog.V(1).Infoln("initial cache sync stopped")
		runtime.HandleError(fmt.Errorf("failed to do initial sync on resources required for ingress"))
		return
	}

	glog.V(1).Infoln("initial cache sync done")
}

// Stop function stops all informers in the context.
func (c *Context) Stop() {
	c.stopChannel <- struct{}{}
}

func isIngressApplicationGateway(ingress *v1beta1.Ingress) bool {
	val, _ := annotations.IsApplicationGatewayIngress(ingress)
	return val
}
