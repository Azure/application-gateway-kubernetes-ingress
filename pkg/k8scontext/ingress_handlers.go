package k8scontext

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// ingress resource handlers
func (h handlers) ingressAdd(obj interface{}) {
	ing := obj.(*v1beta1.Ingress)

	if !IsIngressApplicationGateway(ing) {
		return
	}

	if ing.Spec.TLS != nil && len(ing.Spec.TLS) > 0 {
		ingKey := utils.GetResourceKey(ing.Namespace, ing.Name)
		for _, tls := range ing.Spec.TLS {
			secKey := utils.GetResourceKey(ing.Namespace, tls.SecretName)

			if h.context.ingressSecretsMap.ContainsPair(ingKey, secKey) {
				continue
			}

			if secret, exists, err := h.context.Caches.Secret.GetByKey(secKey); exists && err == nil {
				if !h.context.ingressSecretsMap.ContainsValue(secKey) {
					if err := h.context.CertificateSecretStore.convertSecret(secKey, secret.(*v1.Secret)); err != nil {
						continue
					}
				}
			}

			h.context.ingressSecretsMap.Insert(ingKey, secKey)
		}
	}
	h.context.Work <- events.Event{
		Type:  events.Create,
		Value: obj,
	}
}

func (h handlers) ingressDelete(obj interface{}) {
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
	if !IsIngressApplicationGateway(ing) {
		return
	}
	ingKey := utils.GetResourceKey(ing.Namespace, ing.Name)
	h.context.ingressSecretsMap.Erase(ingKey)

	h.context.Work <- events.Event{
		Type:  events.Delete,
		Value: obj,
	}
}

func (h handlers) ingressUpdate(oldObj, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	oldIng := oldObj.(*v1beta1.Ingress)
	ing := newObj.(*v1beta1.Ingress)
	if !IsIngressApplicationGateway(ing) && !IsIngressApplicationGateway(oldIng) {
		return
	}
	if ing.Spec.TLS != nil && len(ing.Spec.TLS) > 0 {
		ingKey := utils.GetResourceKey(ing.Namespace, ing.Name)
		h.context.ingressSecretsMap.Clear(ingKey)
		for _, tls := range ing.Spec.TLS {
			secKey := utils.GetResourceKey(ing.Namespace, tls.SecretName)

			if h.context.ingressSecretsMap.ContainsPair(ingKey, secKey) {
				continue
			}

			if secret, exists, err := h.context.Caches.Secret.GetByKey(secKey); exists && err == nil {
				if !h.context.ingressSecretsMap.ContainsValue(secKey) {
					if err := h.context.CertificateSecretStore.convertSecret(secKey, secret.(*v1.Secret)); err != nil {
						continue
					}
				}
			}

			h.context.ingressSecretsMap.Insert(ingKey, secKey)
		}
	}

	h.context.Work <- events.Event{
		Type:  events.Update,
		Value: newObj,
	}
}
