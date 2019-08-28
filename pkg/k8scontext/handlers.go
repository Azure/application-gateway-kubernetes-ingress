package k8scontext

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

type handlers struct {
	context *Context
}

// ingress resource handlers
func (h handlers) ingressAddFunc(obj interface{}) {
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
	h.context.UpdateChannel.In() <- events.Event{
		Type:  events.Create,
		Value: obj,
	}
}

func (h handlers) ingressDeleteFunc(obj interface{}) {
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

	h.context.UpdateChannel.In() <- events.Event{
		Type:  events.Delete,
		Value: obj,
	}
}

func (h handlers) ingressUpdateFunc(oldObj, newObj interface{}) {
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

	h.context.UpdateChannel.In() <- events.Event{
		Type:  events.Update,
		Value: newObj,
	}
}

// secret resource handlers
func (h handlers) secretAddFunc(obj interface{}) {
	sec := obj.(*v1.Secret)
	secKey := utils.GetResourceKey(sec.Namespace, sec.Name)
	if h.context.ingressSecretsMap.ContainsValue(secKey) {
		// find if this secKey exists in the map[string]UnorderedSets
		if err := h.context.CertificateSecretStore.convertSecret(secKey, sec); err == nil {
			h.context.UpdateChannel.In() <- events.Event{
				Type:  events.Create,
				Value: obj,
			}
		}
	}
}

func (h handlers) secretUpdateFunc(oldObj, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}

	sec := newObj.(*v1.Secret)
	secKey := utils.GetResourceKey(sec.Namespace, sec.Name)
	if h.context.ingressSecretsMap.ContainsValue(secKey) {
		if err := h.context.CertificateSecretStore.convertSecret(secKey, sec); err == nil {
			h.context.UpdateChannel.In() <- events.Event{
				Type:  events.Update,
				Value: newObj,
			}
		}
	}
}

func (h handlers) secretDeleteFunc(obj interface{}) {
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
	h.context.CertificateSecretStore.eraseSecret(secKey)
	if h.context.ingressSecretsMap.ContainsValue(secKey) {
		h.context.UpdateChannel.In() <- events.Event{
			Type:  events.Delete,
			Value: obj,
		}
	}
}

// general resource handlers
func (h handlers) addFunc(obj interface{}) {
	h.context.UpdateChannel.In() <- events.Event{
		Type:  events.Create,
		Value: obj,
	}
}

func (h handlers) updateFunc(oldObj, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	h.context.UpdateChannel.In() <- events.Event{
		Type:  events.Update,
		Value: newObj,
	}
}

func (h handlers) deleteFunc(obj interface{}) {
	h.context.UpdateChannel.In() <- events.Event{
		Type:  events.Delete,
		Value: obj,
	}
}
