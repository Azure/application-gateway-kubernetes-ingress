// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"fmt"
	"sort"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/glog"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	prohibitedv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/informers/externalversions"
	istio_versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned"
	istio_externalversions "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/informers/externalversions"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

const providerPrefix = "azure://"
const workBuffer = 1024

// NewContext creates a context based on a Kubernetes client instance.
func NewContext(kubeClient kubernetes.Interface, crdClient versioned.Interface, istioCrdClient istio_versioned.Interface, namespaces []string, resyncPeriod time.Duration, metricStore metricstore.MetricStore) *Context {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, resyncPeriod)
	crdInformerFactory := externalversions.NewSharedInformerFactory(crdClient, resyncPeriod)
	istioCrdInformerFactory := istio_externalversions.NewSharedInformerFactoryWithOptions(istioCrdClient, resyncPeriod)

	informerCollection := InformerCollection{
		Endpoints: informerFactory.Core().V1().Endpoints().Informer(),
		Ingress:   informerFactory.Extensions().V1beta1().Ingresses().Informer(),
		Pods:      informerFactory.Core().V1().Pods().Informer(),
		Secret:    informerFactory.Core().V1().Secrets().Informer(),
		Service:   informerFactory.Core().V1().Services().Informer(),

		AzureIngressProhibitedTarget: crdInformerFactory.Azureingressprohibitedtargets().V1().AzureIngressProhibitedTargets().Informer(),

		IstioGateway:        istioCrdInformerFactory.Networking().V1alpha3().Gateways().Informer(),
		IstioVirtualService: istioCrdInformerFactory.Networking().V1alpha3().VirtualServices().Informer(),
	}

	cacheCollection := CacheCollection{
		Endpoints:                    informerCollection.Endpoints.GetStore(),
		Ingress:                      informerCollection.Ingress.GetStore(),
		Pods:                         informerCollection.Pods.GetStore(),
		Secret:                       informerCollection.Secret.GetStore(),
		Service:                      informerCollection.Service.GetStore(),
		AzureIngressProhibitedTarget: informerCollection.AzureIngressProhibitedTarget.GetStore(),
		IstioGateway:                 informerCollection.IstioGateway.GetStore(),
		IstioVirtualService:          informerCollection.IstioVirtualService.GetStore(),
	}

	context := &Context{
		kubeClient:     kubeClient,
		crdClient:      crdClient,
		istioCrdClient: istioCrdClient,

		informers:              &informerCollection,
		ingressSecretsMap:      utils.NewThreadsafeMultimap(),
		Caches:                 &cacheCollection,
		CertificateSecretStore: NewSecretStore(),
		Work:                   make(chan events.Event, workBuffer),
		CacheSynced:            make(chan interface{}),

		metricStore: metricStore,
		namespaces:  make(map[string]interface{}),
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

	return context
}

// Run executes informer collection.
func (c *Context) Run(stopChannel chan struct{}, omitCRDs bool, envVariables environment.EnvVariables) error {
	glog.V(1).Infoln("k8s context run started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return ErrorInformersNotInitialized
	}
	crds := map[cache.SharedInformer]interface{}{
		c.informers.AzureIngressProhibitedTarget: nil,
		c.informers.IstioGateway:                 nil,
		c.informers.IstioVirtualService:          nil,
	}

	sharedInformers := []cache.SharedInformer{
		c.informers.Endpoints,
		c.informers.Pods,
		c.informers.Service,
		c.informers.Secret,
		c.informers.Ingress,
	}

	// For AGIC to watch for these CRDs the EnableBrownfieldDeploymentVarName env variable must be set to true
	if envVariables.EnableBrownfieldDeployment {
		sharedInformers = append(sharedInformers, c.informers.AzureIngressProhibitedTarget)
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

	glog.V(1).Infoln("Waiting for initial cache sync")
	if !cache.WaitForCacheSync(stopChannel, hasSynced...) {
		return ErrorFailedInitialCacheSync
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.CacheSynced)

	glog.V(1).Infoln("Initial cache sync done")
	glog.V(1).Infoln("k8s context run finished")
	return nil
}

// GetAGICPod returns the pod with specified name and namespace
func (c *Context) GetAGICPod(envVariables environment.EnvVariables) *v1.Pod {
	pod, err := c.kubeClient.CoreV1().Pods(envVariables.AGICPodNamespace).Get(envVariables.AGICPodName, metav1.GetOptions{})
	if err != nil {
		glog.Error("Error fetching AGIC Pod (This may happen if AGIC is running in a test environment), error occurred ", err)
		return nil
	}
	return pod
}

// ListServices returns a list of all the Services from cache.
func (c *Context) ListServices() []*v1.Service {
	var serviceList []*v1.Service
	for _, serviceInterface := range c.Caches.Service.List() {
		service := serviceInterface.(*v1.Service)
		if _, exists := c.namespaces[service.Namespace]; len(c.namespaces) > 0 && !exists {
			continue
		}
		if hasTCPPort(service) {
			serviceList = append(serviceList, service)
		}
	}
	return serviceList
}

// GetEndpointsByService returns the endpoints associated with a specific service.
func (c *Context) GetEndpointsByService(serviceKey string) (*v1.Endpoints, error) {
	endpointsInterface, exist, err := c.Caches.Endpoints.GetByKey(serviceKey)

	if err != nil {
		glog.Error("Error fetching endpoints from store, error occurred ", err)
		return nil, err
	}

	if !exist {
		glog.Error("Error fetching endpoints from store! Service does not exist: ", serviceKey)
		return nil, ErrorFetchingEnpdoints
	}

	return endpointsInterface.(*v1.Endpoints), nil
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
func (c *Context) ListHTTPIngresses() []*v1beta1.Ingress {
	var ingressList []*v1beta1.Ingress
	for _, ingressInterface := range c.Caches.Ingress.List() {
		ingress := ingressInterface.(*v1beta1.Ingress)
		if _, exists := c.namespaces[ingress.Namespace]; len(c.namespaces) > 0 && !exists {
			continue
		}
		ingressList = append(ingressList, ingress)
	}
	return filterAndSort(ingressList)
}

func filterAndSort(ingList []*v1beta1.Ingress) []*v1beta1.Ingress {
	var ingressList []*v1beta1.Ingress
	for _, ingress := range ingList {
		if !IsIngressApplicationGateway(ingress) {
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

	glog.V(5).Infof("AzureIngressProhibitedTargets: %+v", strings.Join(prohibitedTargets, ","))

	return targets
}

// GetService returns the service identified by the key.
func (c *Context) GetService(serviceKey string) *v1.Service {
	serviceInterface, exist, err := c.Caches.Service.GetByKey(serviceKey)

	if err != nil {
		glog.V(3).Infof("unable to get service from store, error occurred %s", err)
		return nil
	}

	if !exist {
		glog.V(9).Infof("Service %s does not exist", serviceKey)
		return nil
	}

	service := serviceInterface.(*v1.Service)
	return service
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
	glog.V(5).Infof("Found Virtual Services: %+v", strings.Join(virtualServiceLogging, ","))
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
		if annotated, _ := annotations.IsIstioGatewayIngress(gateway); annotated {
			annotatedGateways = append(annotatedGateways, gateway)
		}
	}
	return annotatedGateways
}

// GetInfrastructureResourceGroupID returns the subscription and resource group name of the underling infrastructure.
// This uses ProviderID which is ID of the node assigned by the cloud provider in the format: <ProviderName>://<ProviderSpecificNodeID>
func (c *Context) GetInfrastructureResourceGroupID() (azure.SubscriptionID, azure.ResourceGroup, error) {
	nodes, err := c.kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return azure.SubscriptionID(""), azure.ResourceGroup(""), err
	}
	if nodes == nil || len(nodes.Items) == 0 {
		return azure.SubscriptionID(""), azure.ResourceGroup(""), ErrorNoNodesFound
	}
	if !strings.HasPrefix(nodes.Items[0].Spec.ProviderID, providerPrefix) {
		return azure.SubscriptionID(""), azure.ResourceGroup(""), ErrorUnrecognizedNodeProviderPrefix
	}
	subscriptionID, resourceGroup, _ := azure.ParseResourceID(strings.TrimPrefix(nodes.Items[0].Spec.ProviderID, providerPrefix))
	return subscriptionID, resourceGroup, nil
}

// UpdateIngressStatus adds IP address in Ingress Status
func (c *Context) UpdateIngressStatus(ingressToUpdate v1beta1.Ingress, newIP IPAddress) error {
	ingressClient := c.kubeClient.ExtensionsV1beta1().Ingresses(ingressToUpdate.Namespace)
	ingress, err := ingressClient.Get(ingressToUpdate.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Unable to get ingress %s/%s", ingressToUpdate.Namespace, ingressToUpdate.Name)
	}

	for _, lbi := range ingress.Status.LoadBalancer.Ingress {
		existingIP := lbi.IP
		if existingIP == string(newIP) {
			glog.V(5).Infof("IP %s already set on Ingress %s/%s", lbi.IP, ingress.Namespace, ingress.Name)
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

	if _, err := ingressClient.UpdateStatus(ingress); err != nil {
		errorLine := fmt.Sprintf("Unable to update ingress %s/%s status: error %s", ingress.Namespace, ingress.Name, err)
		glog.Error(errorLine)
		return ErrorUnableToUpdateIngress
	}

	return nil
}

// IsIngressApplicationGateway checks if applicaiton gateway annotation is present on the ingress
func IsIngressApplicationGateway(ingress *v1beta1.Ingress) bool {
	val, _ := annotations.IsApplicationGatewayIngress(ingress)
	return val
}

func hasHTTPRule(ingress *v1beta1.Ingress) bool {
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
			for _, path := range rule.HTTP.Paths {
				// TODO(akshaysngupta) Use service ports
				if path.Backend.ServiceName == service.Name {
					return true
				}
			}
		}
	}

	return false
}
