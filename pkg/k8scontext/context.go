// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	agpoolv1beta1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewaybackendpool/v1beta1"
	aginstv1beta1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayinstanceupdatestatus/v1beta1"
	agrewritev1beta1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayrewrite/v1beta1"
	prohibitedv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	multiClusterIngress "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/multiclusteringress/v1alpha1"
	multiClusterService "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/multiclusterservice/v1alpha1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/informers/externalversions"
	multicluster_versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned"
	multicluster_externalversions "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/informers/externalversions"
	istio_versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned"
	istio_externalversions "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/informers/externalversions"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext/convert"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

const providerPrefix = "azure://"
const workBuffer = 1024

var namespacesToIgnore = map[string]interface{}{
	"kube-system": nil,
	"kube-public": nil,
}

// NewContext creates a context based on a Kubernetes client instance.
func NewContext(kubeClient kubernetes.Interface, crdClient versioned.Interface, multiClusterCrdClient multicluster_versioned.Interface, istioCrdClient istio_versioned.Interface, namespaces []string, resyncPeriod time.Duration, metricStore metricstore.MetricStore, envVariables environment.EnvVariables) *Context {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, resyncPeriod)
	crdInformerFactory := externalversions.NewSharedInformerFactory(crdClient, resyncPeriod)
	multiClusterCrdInformerFactory := multicluster_externalversions.NewSharedInformerFactory(multiClusterCrdClient, resyncPeriod)
	istioCrdInformerFactory := istio_externalversions.NewSharedInformerFactoryWithOptions(istioCrdClient, resyncPeriod)

	informerCollection := InformerCollection{
		Endpoints: informerFactory.Core().V1().Endpoints().Informer(),
		Pods:      informerFactory.Core().V1().Pods().Informer(),
		Secret:    informerFactory.Core().V1().Secrets().Informer(),
		Service:   informerFactory.Core().V1().Services().Informer(),

		AzureIngressProhibitedTarget:                crdInformerFactory.Azureingressprohibitedtargets().V1().AzureIngressProhibitedTargets().Informer(),
		AzureApplicationGatewayBackendPool:          crdInformerFactory.Azureapplicationgatewaybackendpools().V1beta1().AzureApplicationGatewayBackendPools().Informer(),
		AzureApplicationGatewayRewrite:              crdInformerFactory.Azureapplicationgatewayrewrites().V1beta1().AzureApplicationGatewayRewrites().Informer(),
		AzureApplicationGatewayInstanceUpdateStatus: crdInformerFactory.Azureapplicationgatewayinstanceupdatestatus().V1beta1().AzureApplicationGatewayInstanceUpdateStatuses().Informer(),
		MultiClusterService:                         multiClusterCrdInformerFactory.Multiclusterservices().V1alpha1().MultiClusterServices().Informer(),
		MultiClusterIngress:                         multiClusterCrdInformerFactory.Multiclusteringresses().V1alpha1().MultiClusterIngresses().Informer(),
		IstioGateway:                                istioCrdInformerFactory.Networking().V1alpha3().Gateways().Informer(),
		IstioVirtualService:                         istioCrdInformerFactory.Networking().V1alpha3().VirtualServices().Informer(),
	}

	if IsNetworkingV1PackageSupported {
		informerCollection.Ingress = informerFactory.Networking().V1().Ingresses().Informer()
	} else {
		informerCollection.Ingress = informerFactory.Extensions().V1beta1().Ingresses().Informer()
	}

	cacheCollection := CacheCollection{
		Endpoints:                          informerCollection.Endpoints.GetStore(),
		Ingress:                            informerCollection.Ingress.GetStore(),
		Pods:                               informerCollection.Pods.GetStore(),
		Secret:                             informerCollection.Secret.GetStore(),
		Service:                            informerCollection.Service.GetStore(),
		AzureIngressProhibitedTarget:       informerCollection.AzureIngressProhibitedTarget.GetStore(),
		AzureApplicationGatewayBackendPool: informerCollection.AzureApplicationGatewayBackendPool.GetStore(),
		AzureApplicationGatewayRewrite:     informerCollection.AzureApplicationGatewayRewrite.GetStore(),
		AzureApplicationGatewayInstanceUpdateStatus: informerCollection.AzureApplicationGatewayInstanceUpdateStatus.GetStore(),
		MultiClusterService:                         informerCollection.MultiClusterService.GetStore(),
		MultiClusterIngress:                         informerCollection.MultiClusterIngress.GetStore(),
		IstioGateway:                                informerCollection.IstioGateway.GetStore(),
		IstioVirtualService:                         informerCollection.IstioVirtualService.GetStore(),
	}

	context := &Context{
		kubeClient:            kubeClient,
		crdClient:             crdClient,
		multiClusterCrdClient: multiClusterCrdClient,
		istioCrdClient:        istioCrdClient,

		informers:              &informerCollection,
		ingressSecretsMap:      utils.NewThreadsafeMultimap(),
		Caches:                 &cacheCollection,
		CertificateSecretStore: NewSecretStore(),
		Work:                   make(chan events.Event, workBuffer),
		CacheSynced:            make(chan interface{}),

		MetricStore: metricStore,
		namespaces:  make(map[string]interface{}),

		ingressClassControllerName:  envVariables.IngressClassControllerName,
		ingressClassResourceName:    envVariables.IngressClassResourceName,
		ingressClassResourceEnabled: envVariables.IngressClassResourceEnabled,
		ingressClassResourceDefault: envVariables.IngressClassResourceDefault,
	}

	for _, ns := range namespaces {
		context.namespaces[ns] = nil
	}

	h := handlers{context}

	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.addFunc,
		UpdateFunc: h.updateFunc,
		DeleteFunc: h.deleteFunc,
	}

	ingressResourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.ingressAdd,
		UpdateFunc: h.ingressUpdate,
		DeleteFunc: h.ingressDelete,
	}

	secretResourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.secretAdd,
		UpdateFunc: h.secretUpdate,
		DeleteFunc: h.secretDelete,
	}

	// Register event handlers.
	informerCollection.Endpoints.AddEventHandler(resourceHandler)
	informerCollection.Ingress.AddEventHandler(ingressResourceHandler)
	informerCollection.Pods.AddEventHandler(resourceHandler)
	informerCollection.Secret.AddEventHandler(secretResourceHandler)
	informerCollection.Service.AddEventHandler(resourceHandler)
	informerCollection.AzureIngressProhibitedTarget.AddEventHandler(resourceHandler)
	informerCollection.AzureApplicationGatewayRewrite.AddEventHandler(resourceHandler)
	informerCollection.AzureApplicationGatewayBackendPool.AddEventHandler(resourceHandler)
	informerCollection.AzureApplicationGatewayInstanceUpdateStatus.AddEventHandler(resourceHandler)
	informerCollection.MultiClusterService.AddEventHandler(resourceHandler)
	informerCollection.MultiClusterIngress.AddEventHandler(resourceHandler)

	if IsNetworkingV1PackageSupported {
		informerCollection.IngressClass = informerFactory.Networking().V1().IngressClasses().Informer()
		informerCollection.IngressClass.AddEventHandler(resourceHandler)
		cacheCollection.IngressClass = informerCollection.IngressClass.GetStore()
	}

	return context
}

// Run executes informer collection.
func (c *Context) Run(stopChannel chan struct{}, omitCRDs bool, envVariables environment.EnvVariables) error {
	klog.V(1).Infoln("k8s context run started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		e := controllererrors.NewError(
			controllererrors.ErrorInformersNotInitialized,
			"informers are not initialized",
		)
		c.MetricStore.IncErrorCount(e.Code)
		return e
	}
	crds := map[cache.SharedInformer]interface{}{
		c.informers.AzureIngressProhibitedTarget: nil,
		c.informers.IstioGateway:                 nil,
		c.informers.IstioVirtualService:          nil,
		c.informers.MultiClusterService:          nil,
		c.informers.MultiClusterIngress:          nil,

		c.informers.AzureApplicationGatewayRewrite: nil,
		// c.informers.AzureApplicationGatewayBackendPool:          nil,
		// c.informers.AzureApplicationGatewayInstanceUpdateStatus: nil,
	}

	sharedInformers := []cache.SharedInformer{
		c.informers.Endpoints,
		c.informers.Pods,
		c.informers.Service,
		c.informers.Secret,
		c.informers.Ingress,

		c.informers.AzureApplicationGatewayRewrite,

		//TODO: enabled by ccp feature flag
		// c.informers.AzureApplicationGatewayBackendPool,
		// c.informers.AzureApplicationGatewayInstanceUpdateStatus,
	}

	if IsNetworkingV1PackageSupported {
		sharedInformers = append(sharedInformers, c.informers.IngressClass)
	}

	// For AGIC to watch for these CRDs the EnableBrownfieldDeploymentVarName env variable must be set to true
	if envVariables.EnableBrownfieldDeployment {
		sharedInformers = append(sharedInformers, c.informers.AzureIngressProhibitedTarget)
	}

	// For AGIC to watch for these CRDs the MultiClusterMode env variable must be set to true
	if envVariables.MultiClusterMode {
		sharedInformers = []cache.SharedInformer{} //only need to monitor 3 resources
		sharedInformers = append(sharedInformers, c.informers.MultiClusterIngress)
		sharedInformers = append(sharedInformers, c.informers.MultiClusterService)
	}

	if envVariables.EnableIstioIntegration {
		sharedInformers = append(sharedInformers, c.informers.IstioGateway, c.informers.IstioVirtualService)
	}

	for _, informer := range sharedInformers {
		go informer.Run(stopChannel)
		// NOTE: Delyan could not figure out how to make informer.HasSynced == true for the CRDs in unit tests
		// so until we do that - we omit WaitForCacheSync for CRDs in unit testing
		if _, isCRD := crds[informer]; isCRD {
			continue
		}
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	klog.V(1).Infoln("Waiting for initial cache sync")
	if !cache.WaitForCacheSync(stopChannel, hasSynced...) {
		e := controllererrors.NewError(
			controllererrors.ErrorFailedInitialCacheSync,
			"failed initial sync of resources required for ingress",
		)
		c.MetricStore.IncErrorCount(e.Code)
		return e
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.CacheSynced)

	klog.V(1).Infoln("Initial cache sync done")
	klog.V(1).Infoln("k8s context run finished")
	return nil
}

// GetAGICPod returns the pod with specified name and namespace
func (c *Context) GetAGICPod(envVariables environment.EnvVariables) *v1.Pod {
	pod, err := c.kubeClient.CoreV1().Pods(envVariables.AGICPodNamespace).Get(context.TODO(), envVariables.AGICPodName, metav1.GetOptions{})
	if err != nil {
		klog.Error("Error fetching AGIC Pod (This may happen if AGIC is running in a test environment). Error: ", err)
		return nil
	}
	return pod
}

// GetBackendPool returns backend pool with specified name
func (c *Context) GetBackendPool(backendPoolName string) (*agpoolv1beta1.AzureApplicationGatewayBackendPool, error) {
	agpool, exist, err := c.Caches.AzureApplicationGatewayBackendPool.GetByKey(backendPoolName)
	if !exist {
		e := controllererrors.NewErrorf(
			controllererrors.ErrorFetchingBackendAddressPool,
			"Backend pool CRD object not found for %s",
			backendPoolName)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}

	if err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorFetchingBackendAddressPool,
			err,
			"Error fetching backend pool CRD object from store for %s",
			backendPoolName)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}

	return agpool.(*agpoolv1beta1.AzureApplicationGatewayBackendPool), nil
}

// GetRewriteRuleSetCustomResource returns rewrite with specified name and namespace
func (c *Context) GetRewriteRuleSetCustomResource(namespace string, name string) (*agrewritev1beta1.AzureApplicationGatewayRewrite, error) {

	agrewrite, exist, err := c.Caches.AzureApplicationGatewayRewrite.GetByKey(namespace + "/" + name)
	if !exist {
		e := controllererrors.NewErrorf(
			controllererrors.ErrorFetchingRewrite,
			"Rewrite rule set custom resource object not found for %s",
			name)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}

	if err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorFetchingRewrite,
			err,
			"Error fetching rewrite rule set custom resource object from store for %s",
			name)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}

	return agrewrite.(*agrewritev1beta1.AzureApplicationGatewayRewrite), nil
}

// GetInstanceUpdateStatus returns update status from when Application Gateway instances update backend pool addresses
func (c *Context) GetInstanceUpdateStatus(instanceUpdateStatusName string) (*aginstv1beta1.AzureApplicationGatewayInstanceUpdateStatus, error) {
	agpool, exist, err := c.Caches.AzureApplicationGatewayInstanceUpdateStatus.GetByKey(instanceUpdateStatusName)
	if !exist {
		e := controllererrors.NewErrorf(
			controllererrors.ErrorFetchingInstanceUpdateStatus,
			"Instance update status CRD object not found for %s",
			instanceUpdateStatusName)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}

	if err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorFetchingInstanceUpdateStatus,
			err,
			"Error fetching instance update status CRD object from store for %s",
			instanceUpdateStatusName)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}

	return agpool.(*aginstv1beta1.AzureApplicationGatewayInstanceUpdateStatus), nil
}

// GetProhibitedTarget returns prohibited target with specified name and namespace
func (c *Context) GetProhibitedTarget(namespace string, targetName string) *prohibitedv1.AzureIngressProhibitedTarget {
	target, err := c.crdClient.AzureingressprohibitedtargetsV1().AzureIngressProhibitedTargets(namespace).Get(context.TODO(), targetName, metav1.GetOptions{})
	if err != nil {
		klog.Error("Error fetching Azure ingress prohibired target resource, Error: ", err)
		return nil
	}
	return target
}

// ListServices returns a list of all the Services from cache.
func (c *Context) ListServices() []*v1.Service {
	var serviceList []*v1.Service
	if IsInMultiClusterMode {
		for _, multiClusterServiceInterface := range c.Caches.MultiClusterService.List() {
			multiClusterService := multiClusterServiceInterface.(*multiClusterService.MultiClusterService)
			service, exists := convert.FromMultiClusterService(multiClusterService)
			if !exists {
				klog.Error("Unable to convert MultiClusterService to Service")
				continue
			}
			if _, exists := c.namespaces[service.Namespace]; len(c.namespaces) > 0 && !exists {
				continue
			}
			if hasTCPPort(service) {
				serviceList = append(serviceList, service)
			}
		}
	} else {
		for _, serviceInterface := range c.Caches.Service.List() {
			service := serviceInterface.(*v1.Service)
			if _, exists := c.namespaces[service.Namespace]; len(c.namespaces) > 0 && !exists {
				continue
			}
			if hasTCPPort(service) {
				serviceList = append(serviceList, service)
			}
		}
	}
	return serviceList
}

// GetEndpointsByService returns the endpoints associated with a specific service.
func (c *Context) GetEndpointsByService(serviceKey string) (*v1.Endpoints, error) {
	if IsInMultiClusterMode {
		return c.generateEndpointsFromMultiClusterService(serviceKey)
	}
	endpointsInterface, exist, err := c.Caches.Endpoints.GetByKey(serviceKey)

	if !exist {
		e := controllererrors.NewErrorf(
			controllererrors.ErrorFetchingEndpoints,
			"Endpoint not found for %s",
			serviceKey)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}

	if err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorFetchingEndpoints,
			err,
			"Error fetching endpoints from store for %s",
			serviceKey)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}
	return endpointsInterface.(*v1.Endpoints), nil

}

func (c *Context) generateEndpointsFromMultiClusterService(serviceKey string) (*v1.Endpoints, error) {
	multiClusterServiceInterface, exist, err := c.Caches.MultiClusterService.GetByKey(serviceKey)

	if !exist {
		e := controllererrors.NewErrorf(
			controllererrors.ErrorFetchingMultiClusterService,
			"MultiCluster service not found for %s",
			serviceKey)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}

	if err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorFetchingMultiClusterService,
			err,
			"Error fetching MultiCluster service from store for %s",
			serviceKey)
		klog.Error(e.Error())
		c.MetricStore.IncErrorCount(e.Code)
		return nil, e
	}

	multiClusterService := multiClusterServiceInterface.(*multiClusterService.MultiClusterService)
	endpoints := &v1.Endpoints{}
	subset := v1.EndpointSubset{}

	for _, multiClusterEndpoint := range multiClusterService.Status.Endpoints {
		address := v1.EndpointAddress{IP: multiClusterEndpoint.IP}
		subset.Addresses = append(subset.Addresses, address)
	}

	for _, ports := range multiClusterService.Spec.Ports {
		v1Port := v1.EndpointPort{Port: int32(ports.Port), Protocol: v1.Protocol(ports.Protocol)}
		subset.Ports = append(subset.Ports, v1Port)
	}

	endpoints.Subsets = []v1.EndpointSubset{subset}
	return endpoints, nil
}

// ListPodsByServiceSelector returns pods that are associated with a specific service.
func (c *Context) ListPodsByServiceSelector(service *v1.Service) []*v1.Pod {
	selectorSet := mapset.NewSet()
	for k, v := range service.Spec.Selector {
		selectorSet.Add(k + ":" + v)
	}

	var podList []*v1.Pod
	for _, podInterface := range c.Caches.Pods.List() {
		pod := podInterface.(*v1.Pod)
		if _, exists := c.namespaces[pod.Namespace]; len(c.namespaces) > 0 && !exists {
			continue
		}
		podLabelSet := mapset.NewSet()
		for k, v := range pod.Labels {
			podLabelSet.Add(k + ":" + v)
		}

		if selectorSet.IsSubset(podLabelSet) && pod.Namespace == service.Namespace {
			podList = append(podList, pod)
		}
	}

	return podList
}

// IsPodReferencedByAnyIngress provides whether a POD is useful i.e. a POD is used by an ingress
func (c *Context) IsPodReferencedByAnyIngress(pod *v1.Pod) bool {
	// first find all the services
	services := c.listServicesByPodSelector(pod)
	for _, service := range services {
		if c.isServiceReferencedByAnyIngress(service) {
			return true
		}
	}

	return false
}

// IsEndpointReferencedByAnyIngress provides whether an Endpoint is useful i.e. a Endpoint is used by an ingress
func (c *Context) IsEndpointReferencedByAnyIngress(endpoints *v1.Endpoints) bool {
	service := c.GetService(fmt.Sprintf("%v/%v", endpoints.Namespace, endpoints.Name))
	return service != nil && c.isServiceReferencedByAnyIngress(service)
}

// ListHTTPIngresses returns a list of all the ingresses for HTTP from cache.
func (c *Context) ListHTTPIngresses() []*networking.Ingress {
	var ingressList []*networking.Ingress
	if IsInMultiClusterMode {
		klog.V(9).Infof("Fetching MultiCluster Ingresses")
		for _, mciInterface := range c.Caches.MultiClusterIngress.List() {
			mci := mciInterface.(*multiClusterIngress.MultiClusterIngress)
			ingress, exists := convert.FromMultiClusterIngress(mci)
			if !exists {
				klog.Error("Unable to convert MultiClusterIngress to Ingress")
				continue
			}
			if _, exists := c.namespaces[ingress.Namespace]; len(c.namespaces) > 0 && !exists {
				continue
			}
			ingressList = append(ingressList, ingress)
		}
	} else {
		for _, ingressInterface := range c.Caches.Ingress.List() {
			ingress, _ := convert.ToIngressV1(ingressInterface)
			if _, exists := c.namespaces[ingress.Namespace]; len(c.namespaces) > 0 && !exists {
				continue
			}
			ingressList = append(ingressList, ingress)
		}
	}
	return c.filterAndSort(ingressList)
}

func (c *Context) filterAndSort(ingList []*networking.Ingress) []*networking.Ingress {
	var ingressList []*networking.Ingress
	for _, ingress := range ingList {
		if !c.IsIngressClass(ingress) {
			continue
		}
		if len(ingress.Spec.Rules) > 0 && !hasHTTPRule(ingress) {
			continue
		}
		ingressList = append(ingressList, ingress)
	}
	// Sorting the return list ensures that the iterations over this list and
	// subsequently created structs have deterministic order. This increases
	// cache hits, and lowers the load on ARM.
	sort.Sort(sorter.ByIngressName(ingressList))
	return ingressList
}

// ListAzureProhibitedTargets returns a list of App Gwy configs, for which AGIC is not allowed to modify config.
func (c *Context) ListAzureProhibitedTargets() []*prohibitedv1.AzureIngressProhibitedTarget {
	var targets []*prohibitedv1.AzureIngressProhibitedTarget
	for _, obj := range c.Caches.AzureIngressProhibitedTarget.List() {
		prohibitedTarget := obj.(*prohibitedv1.AzureIngressProhibitedTarget)
		if _, exists := c.namespaces[prohibitedTarget.Namespace]; len(c.namespaces) > 0 && !exists {
			continue
		}
		targets = append(targets, prohibitedTarget)
	}

	var prohibitedTargets []string
	for _, target := range targets {
		prohibitedTargets = append(prohibitedTargets, fmt.Sprintf("%s/%s", target.Namespace, target.Name))
	}

	klog.V(5).Infof("AzureIngressProhibitedTargets: %+v", strings.Join(prohibitedTargets, ","))

	return targets
}

// GetService returns the service identified by the key.
func (c *Context) GetService(serviceKey string) *v1.Service {
	if IsInMultiClusterMode {
		serviceInterface, exist, err := c.Caches.MultiClusterService.GetByKey(serviceKey)

		if err != nil {
			klog.V(3).Infof("unable to get multicluster service from store, error occurred %s", err)
			return nil
		}

		if !exist {
			klog.V(9).Infof("MultiCluster Service %s does not exist", serviceKey)
			return nil
		}

		multiClusterService := serviceInterface.(*multiClusterService.MultiClusterService)
		service, exists := convert.FromMultiClusterService(multiClusterService)

		if !exists {
			klog.V(3).Infof("unable to convert multicluster service from store to service")
			return nil
		}
		return service
	}
	serviceInterface, exist, err := c.Caches.Service.GetByKey(serviceKey)

	if err != nil {
		klog.V(3).Infof("unable to get service from store, error occurred %s", err)
		return nil
	}

	if !exist {
		klog.V(9).Infof("Service %s does not exist", serviceKey)
		return nil
	}

	service := serviceInterface.(*v1.Service)
	return service
}

// GetSecret returns the secret identified by the key
func (c *Context) GetSecret(secretKey string) *v1.Secret {
	secretInterface, exist, err := c.Caches.Secret.GetByKey(secretKey)

	if err != nil {
		klog.Error("Error fetching secret from store:", err)
		return nil
	}

	if !exist {
		klog.Error("Error fetching secret from store! Service does not exist:", secretKey)
		return nil
	}

	secret := secretInterface.(*v1.Secret)
	return secret
}

// GetVirtualServicesForGateway returns the VirtualServices for the provided gateway
func (c *Context) GetVirtualServicesForGateway(gateway v1alpha3.Gateway) []*v1alpha3.VirtualService {
	virtualServices := make([]*v1alpha3.VirtualService, 0)
	allVirtualServices := c.ListIstioVirtualServices()
	gatewayName := gateway.Name
	for _, service := range allVirtualServices {
		hasGateway := false
		for _, serviceGateway := range service.Spec.Gateways {
			if gatewayName == serviceGateway {
				hasGateway = true
			}
		}
		if hasGateway {
			virtualServices = append(virtualServices, service)
		}
	}
	var virtualServiceLogging []string
	for _, virtualService := range virtualServices {
		virtualServiceLogging = append(virtualServiceLogging, fmt.Sprintf("%s/%s", virtualService.Namespace, virtualService.Name))
	}
	klog.V(5).Infof("Found Virtual Services: %+v", strings.Join(virtualServiceLogging, ","))
	return virtualServices
}

// GetEndpointsForVirtualService returns a list of Endpoints associated with a Virtual Service
func (c *Context) GetEndpointsForVirtualService(virtualService v1alpha3.VirtualService) v1.EndpointsList {
	endpointList := make([]v1.Endpoints, 0)
	namespace := virtualService.Namespace
	for _, httpRouteRule := range virtualService.Spec.HTTP {
		for _, route := range httpRouteRule.Route {
			serviceKey := fmt.Sprintf("%v/%v", namespace, route.Destination.Host)
			endpoint, err := c.GetEndpointsByService(serviceKey)
			if err == nil {
				endpointList = append(endpointList, *endpoint)
			}
		}
	}
	return v1.EndpointsList{
		Items: endpointList,
	}
}

// GetGateways returns all Istio Gateways that are annotated.
func (c *Context) GetGateways() []*v1alpha3.Gateway {
	annotatedGateways := make([]*v1alpha3.Gateway, 0)
	for _, gateway := range c.ListIstioGateways() {
		if annotated := c.IsIstioGatewayIngress(gateway); annotated {
			annotatedGateways = append(annotatedGateways, gateway)
		}
	}
	return annotatedGateways
}

// GetInfrastructureResourceGroupID returns the subscription and resource group name of the underling infrastructure.
// This uses ProviderID which is ID of the node assigned by the cloud provider in the format: <ProviderName>://<ProviderSpecificNodeID>
func (c *Context) GetInfrastructureResourceGroupID() (azure.SubscriptionID, azure.ResourceGroup, error) {
	nodes, err := c.kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		e := controllererrors.NewErrorWithInnerError(
			controllererrors.ErrorFetchingNodes,
			err,
			"no nodes were found in the node list",
		)
		c.MetricStore.IncErrorCount(e.Code)
		return azure.SubscriptionID(""), azure.ResourceGroup(""), e
	}
	if nodes == nil || len(nodes.Items) == 0 {
		e := controllererrors.NewError(
			controllererrors.ErrorNoNodesFound,
			"no nodes were found in the node list",
		)
		c.MetricStore.IncErrorCount(e.Code)
		return azure.SubscriptionID(""), azure.ResourceGroup(""), e
	}
	if !strings.HasPrefix(nodes.Items[0].Spec.ProviderID, providerPrefix) {
		e := controllererrors.NewError(
			controllererrors.ErrorUnrecognizedNodeProviderPrefix,
			"providerID is not prefixed with azure://",
		)
		c.MetricStore.IncErrorCount(e.Code)
		return azure.SubscriptionID(""), azure.ResourceGroup(""), e
	}
	subscriptionID, resourceGroup, _ := azure.ParseResourceID(strings.TrimPrefix(nodes.Items[0].Spec.ProviderID, providerPrefix))
	return subscriptionID, resourceGroup, nil
}

// UpdateIngressStatus adds IP address in Ingress Status
func (c *Context) UpdateIngressStatus(ingressToUpdate networking.Ingress, newIP IPAddress) error {
	if IsNetworkingV1PackageSupported && !IsInMultiClusterMode {
		return c.updateV1IngressStatus(ingressToUpdate, newIP)
	} else if IsInMultiClusterMode {
		return c.updateMultiClusterIngressStatus(ingressToUpdate, newIP)
	} else {
		return c.updateV1beta1IngressStatus(ingressToUpdate, newIP)
	}

}

func (c *Context) updateV1IngressStatus(ingressToUpdate networking.Ingress, newIP IPAddress) error {
	ingressClient := c.kubeClient.NetworkingV1().Ingresses(ingressToUpdate.Namespace)
	ingress, err := ingressClient.Get(context.TODO(), ingressToUpdate.Name, metav1.GetOptions{})
	if err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorUpdatingIngressStatus,
			err,
			"Unable to get ingress %s/%s", ingressToUpdate.Namespace, ingressToUpdate.Name,
		)
		c.MetricStore.IncErrorCount(e.Code)
		return e
	}

	for _, lbi := range ingress.Status.LoadBalancer.Ingress {
		existingIP := lbi.IP
		if existingIP == string(newIP) {
			klog.Infof("IP %s already set on Ingress %s/%s", lbi.IP, ingress.Namespace, ingress.Name)
			return nil
		}
	}

	loadBalancerIngresses := []v1.LoadBalancerIngress{}
	if newIP != "" {
		loadBalancerIngresses = append(loadBalancerIngresses, v1.LoadBalancerIngress{
			IP: string(newIP),
		})
	}
	ingress.Status.LoadBalancer.Ingress = loadBalancerIngresses

	if _, err := ingressClient.UpdateStatus(context.TODO(), ingress, metav1.UpdateOptions{}); err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorUpdatingIngressStatus,
			err,
			"Unable to update ingress %s/%s status", ingress.Namespace, ingress.Name,
		)
		c.MetricStore.IncErrorCount(e.Code)
		return e
	}

	return nil
}

func (c *Context) updateV1beta1IngressStatus(ingressToUpdate networking.Ingress, newIP IPAddress) error {
	ingressClient := c.kubeClient.ExtensionsV1beta1().Ingresses(ingressToUpdate.Namespace)
	ingress, err := ingressClient.Get(context.TODO(), ingressToUpdate.Name, metav1.GetOptions{})
	if err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorUpdatingIngressStatus,
			err,
			"Unable to get ingress %s/%s", ingressToUpdate.Namespace, ingressToUpdate.Name,
		)
		c.MetricStore.IncErrorCount(e.Code)
		return e
	}

	for _, lbi := range ingress.Status.LoadBalancer.Ingress {
		existingIP := lbi.IP
		if existingIP == string(newIP) {
			klog.Infof("IP %s already set on Ingress %s/%s", lbi.IP, ingress.Namespace, ingress.Name)
			return nil
		}
	}

	loadBalancerIngresses := []v1.LoadBalancerIngress{}
	if newIP != "" {
		loadBalancerIngresses = append(loadBalancerIngresses, v1.LoadBalancerIngress{
			IP: string(newIP),
		})
	}
	ingress.Status.LoadBalancer.Ingress = loadBalancerIngresses

	if _, err := ingressClient.UpdateStatus(context.TODO(), ingress, metav1.UpdateOptions{}); err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorUpdatingIngressStatus,
			err,
			"Unable to update ingress %s/%s status", ingress.Namespace, ingress.Name,
		)
		c.MetricStore.IncErrorCount(e.Code)
		return e
	}
	return nil
}

func (c *Context) updateMultiClusterIngressStatus(ingressToUpdate networking.Ingress, newIP IPAddress) error {
	ingressClient := c.multiClusterCrdClient.MulticlusteringressesV1alpha1().MultiClusterIngresses(ingressToUpdate.Namespace)
	ingress, err := ingressClient.Get(context.TODO(), ingressToUpdate.Name, metav1.GetOptions{})
	if err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorUpdatingIngressStatus,
			err,
			"Unable to get ingress %s/%s", ingressToUpdate.Namespace, ingressToUpdate.Name,
		)
		c.MetricStore.IncErrorCount(e.Code)
		return e
	}

	for _, lbi := range ingress.Status.LoadBalancer.Ingress {
		existingIP := lbi.IP
		if existingIP == string(newIP) {
			klog.Infof("IP %s already set on Ingress %s/%s", lbi.IP, ingress.Namespace, ingress.Name)
			return nil
		}
	}

	loadBalancerIngresses := []v1.LoadBalancerIngress{}
	if newIP != "" {
		loadBalancerIngresses = append(loadBalancerIngresses, v1.LoadBalancerIngress{
			IP: string(newIP),
		})
	}
	ingress.Status.LoadBalancer.Ingress = loadBalancerIngresses

	if _, err := ingressClient.UpdateStatus(context.TODO(), ingress, metav1.UpdateOptions{}); err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorUpdatingIngressStatus,
			err,
			"Unable to update ingress %s/%s status", ingress.Namespace, ingress.Name,
		)
		c.MetricStore.IncErrorCount(e.Code)
		return e
	}
	return nil
}

func hasHTTPRule(ingress *networking.Ingress) bool {
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP != nil {
			return true
		}
	}
	return false
}

func hasTCPPort(service *v1.Service) bool {
	for _, port := range service.Spec.Ports {
		if port.Protocol == v1.ProtocolTCP {
			return true
		}
	}
	return false
}

func (c *Context) listServicesByPodSelector(pod *v1.Pod) []*v1.Service {
	labelSet := mapset.NewSet()
	for k, v := range pod.Labels {
		labelSet.Add(k + ":" + v)
	}

	var serviceList []*v1.Service
	for _, service := range c.ListServices() {
		serviceLabelSet := mapset.NewSet()
		for k, v := range service.Spec.Selector {
			serviceLabelSet.Add(k + ":" + v)
		}

		if serviceLabelSet.IsSubset(labelSet) {
			serviceList = append(serviceList, service)
		}
	}

	return serviceList
}

func (c *Context) isServiceReferencedByAnyIngress(service *v1.Service) bool {
	for _, ingress := range c.ListHTTPIngresses() {
		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				// TODO(akshaysngupta) Use service ports
				if path.Backend.Service.Name == service.Name {
					return true
				}
			}
		}
	}

	return false
}

func (c *Context) getIngressClassResource(ingressClassName string) *networking.IngressClass {
	if class := c.getIngressClassResourceFromCache(ingressClassName); class != nil {
		return class
	}

	return c.getIngressClassResourceFromCluster(ingressClassName)
}

// getIngressClassResource gets ingress class object with specified name
func (c *Context) getIngressClassResourceFromCache(ingressClassName string) *networking.IngressClass {
	if c.Caches.IngressClass == nil {
		return nil
	}

	ingressClassInterface, exist, err := c.Caches.IngressClass.GetByKey(ingressClassName)
	if !exist {
		return nil
	}

	if err != nil {
		klog.Errorf("Unable to fetch IngressClass '%s' from cache. Error: %s", ingressClassName, err)
		return nil
	}

	return ingressClassInterface.(*networking.IngressClass)
}

func (c *Context) getIngressClassResourceFromCluster(ingressClassName string) *networking.IngressClass {
	ingressClass, err := c.kubeClient.NetworkingV1().IngressClasses().Get(context.TODO(), ingressClassName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Unable to fetch IngressClass '%s' from cluster. Error: %s", ingressClassName, err)
		return nil
	}

	return ingressClass
}

// IsIngressClass checks if the Ingress resource can be handled by the Application Gateway ingress controller.
func (c *Context) IsIngressClass(ing *networking.Ingress) bool {

	// match by annotation (for Backward compatibility)
	if className, err := annotations.IngressClass(ing); err == nil && className != "" {
		return className == c.ingressClassControllerName
	}

	// match by ingress class resource
	if c.ingressClassResourceEnabled {
		// if IngressClassName in ingress that compare it with the controller type
		if ing.Spec.IngressClassName != nil {
			ingressClass := c.getIngressClassResource(*ing.Spec.IngressClassName)
			return ingressClass != nil && ingressClass.Spec.Controller == c.ingressClassControllerName &&
				c.ingressClassResourceName == *ing.Spec.IngressClassName
		}

		// if IngressClassName is nil, then match if AGIC is default ingress
		return c.ingressClassResourceDefault
	}

	return false
}

// IsIstioGatewayIngress checks if this gateway should be handled by AGIC or not
func (c *Context) IsIstioGatewayIngress(gateway *v1alpha3.Gateway) bool {
	className, err := annotations.IstioGatewayIngressClass(gateway)
	if err != nil {
		return false
	}

	return className == c.ingressClassControllerName
}
