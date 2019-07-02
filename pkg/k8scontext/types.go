package k8scontext

import (
	"github.com/eapache/channels"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// InformerCollection : all the informers for k8s resources we care about.
type InformerCollection struct {
	Endpoints                      cache.SharedIndexInformer
	Ingress                        cache.SharedIndexInformer
	Pods                           cache.SharedIndexInformer
	Secret                         cache.SharedIndexInformer
	Service                        cache.SharedIndexInformer
	Namespace                      cache.SharedIndexInformer
	AzureIngressManagedLocation    cache.SharedInformer
	AzureIngressProhibitedLocation cache.SharedInformer
	IstioGateway                   cache.SharedIndexInformer
	IstioVirtualService			   cache.SharedIndexInformer
}

// CacheCollection : all the listers from the informers.
type CacheCollection struct {
	Endpoints                      cache.Store
	Ingress                        cache.Store
	Pods                           cache.Store
	Secret                         cache.Store
	Service                        cache.Store
	Namespaces                     cache.Store
	AzureIngressManagedLocation    cache.Store
	AzureIngressProhibitedLocation cache.Store
	IstioGateway                   cache.Store
}

// Context : cache and listener for k8s resources.
type Context struct {
	informers              *InformerCollection
	Caches                 *CacheCollection
	CertificateSecretStore SecretsKeeper

	ingressSecretsMap utils.ThreadsafeMultiMap
	stopChannel       chan struct{}

	UpdateChannel *channels.RingChannel
}
