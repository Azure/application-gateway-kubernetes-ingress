package k8scontext

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	multicluster_versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned"
	istio_versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// InformerCollection : all the informers for k8s resources we care about.
type InformerCollection struct {
	Endpoints                                   cache.SharedIndexInformer
	Ingress                                     cache.SharedIndexInformer
	IngressClass                                cache.SharedIndexInformer
	Pods                                        cache.SharedIndexInformer
	Secret                                      cache.SharedIndexInformer
	Service                                     cache.SharedIndexInformer
	Namespace                                   cache.SharedIndexInformer
	AzureIngressManagedLocation                 cache.SharedInformer
	AzureIngressProhibitedTarget                cache.SharedInformer
	AzureApplicationGatewayBackendPool          cache.SharedInformer
	AzureApplicationGatewayHeaderRewrite        cache.SharedInformer
	AzureApplicationGatewayInstanceUpdateStatus cache.SharedInformer
	MultiClusterService                         cache.SharedInformer
	MultiClusterIngress                         cache.SharedInformer
	IstioGateway                                cache.SharedIndexInformer
	IstioVirtualService                         cache.SharedIndexInformer
}

// CacheCollection : all the listers from the informers.
type CacheCollection struct {
	Endpoints                                   cache.Store
	Ingress                                     cache.Store
	IngressClass                                cache.Store
	Pods                                        cache.Store
	Secret                                      cache.Store
	Service                                     cache.Store
	Namespaces                                  cache.Store
	AzureIngressManagedLocation                 cache.Store
	AzureIngressProhibitedTarget                cache.Store
	AzureApplicationGatewayBackendPool          cache.Store
	AzureApplicationGatewayHeaderRewrite        cache.Store
	AzureApplicationGatewayInstanceUpdateStatus cache.Store
	MultiClusterService                         cache.Store
	MultiClusterIngress                         cache.Store
	IstioGateway                                cache.Store
	IstioVirtualService                         cache.Store
}

// Context : cache and listener for k8s resources.
type Context struct {
	// k8s Clients
	kubeClient            kubernetes.Interface
	crdClient             versioned.Interface
	istioCrdClient        istio_versioned.Interface
	multiClusterCrdClient multicluster_versioned.Interface

	informers              *InformerCollection
	Caches                 *CacheCollection
	CertificateSecretStore SecretsKeeper

	ingressSecretsMap utils.ThreadsafeMultiMap

	Work chan events.Event

	CacheSynced chan interface{}

	MetricStore metricstore.MetricStore
	namespaces  map[string]interface{}

	ingressClassControllerName  string
	ingressClassResourceName    string
	ingressClassResourceEnabled bool
	ingressClassResourceDefault bool
}

// IPAddress is type for IP address string
type IPAddress string
