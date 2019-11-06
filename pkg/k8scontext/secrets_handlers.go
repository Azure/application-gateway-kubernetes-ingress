// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// secret resource handlers
func (h handlers) secretAdd(obj interface{}) {
	sec := obj.(*v1.Secret)
	if _, exists := namespacesToIgnore[sec.Namespace]; exists {
		return
	}
	if _, exists := h.context.namespaces[sec.Namespace]; len(h.context.namespaces) > 0 && !exists {
		return
	}

	secKey := utils.GetResourceKey(sec.Namespace, sec.Name)
	if h.context.ingressSecretsMap.ContainsValue(secKey) {
		// find if this secKey exists in the map[string]UnorderedSets
		if err := h.context.CertificateSecretStore.ConvertSecret(secKey, sec); err == nil {
			h.context.Work <- events.Event{
				Type:  events.Create,
				Value: obj,
			}
			h.context.metricStore.IncK8sAPIEventCounter()
		}
	}
}

func (h handlers) secretUpdate(oldObj, newObj interface{}) {
	sec := newObj.(*v1.Secret)
	if _, exists := namespacesToIgnore[sec.Namespace]; exists {
		return
	}
	if _, exists := h.context.namespaces[sec.Namespace]; len(h.context.namespaces) > 0 && !exists {
		return
	}

	if reflect.DeepEqual(oldObj, newObj) {
		return
	}

	secKey := utils.GetResourceKey(sec.Namespace, sec.Name)
	if h.context.ingressSecretsMap.ContainsValue(secKey) {
		if err := h.context.CertificateSecretStore.ConvertSecret(secKey, sec); err == nil {
			h.context.Work <- events.Event{
				Type:  events.Update,
				Value: newObj,
			}
			h.context.metricStore.IncK8sAPIEventCounter()
		}
	}
}

func (h handlers) secretDelete(obj interface{}) {
	sec, ok := obj.(*v1.Secret)
	if _, exists := namespacesToIgnore[sec.Namespace]; exists {
		return
	}
	if _, exists := h.context.namespaces[sec.Namespace]; len(h.context.namespaces) > 0 && !exists {
		return
	}

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
	h.context.CertificateSecretStore.delete(secKey)
	if h.context.ingressSecretsMap.ContainsValue(secKey) {
		h.context.Work <- events.Event{
			Type:  events.Delete,
			Value: obj,
		}
		h.context.metricStore.IncK8sAPIEventCounter()
	}
}
