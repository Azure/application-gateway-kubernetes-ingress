package k8scontext

import (
	"reflect"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// ingress resource handlers
func (h handlers) ingressAdd(obj interface{}) {
	ing := obj.(*v1beta1.Ingress)
	if _, exists := namespacesToIgnore[ing.Namespace]; exists {
		return
	}
	if _, exists := h.context.namespaces[ing.Namespace]; len(h.context.namespaces) > 0 && !exists {
		return
	}

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
					if err := h.context.CertificateSecretStore.ConvertSecret(secKey, secret.(*v1.Secret)); err != nil {
						glog.Error(err.Error())
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
	h.context.MetricStore.IncK8sAPIEventCounter()
}

func (h handlers) ingressDelete(obj interface{}) {
	ing, ok := obj.(*v1beta1.Ingress)
	if _, exists := namespacesToIgnore[ing.Namespace]; exists {
		return
	}
	if _, exists := h.context.namespaces[ing.Namespace]; len(h.context.namespaces) > 0 && !exists {
		return
	}

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
	h.context.MetricStore.IncK8sAPIEventCounter()
}

func (h handlers) ingressUpdate(oldObj, newObj interface{}) {
	ing := newObj.(*v1beta1.Ingress)
	if _, exists := namespacesToIgnore[ing.Namespace]; exists {
		return
	}
	if _, exists := h.context.namespaces[ing.Namespace]; len(h.context.namespaces) > 0 && !exists {
		return
	}

	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	oldIng := oldObj.(*v1beta1.Ingress)
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
					if err := h.context.CertificateSecretStore.ConvertSecret(secKey, secret.(*v1.Secret)); err != nil {
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
	h.context.MetricStore.IncK8sAPIEventCounter()
}
