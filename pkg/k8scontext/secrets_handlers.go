// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"reflect"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// secret resource handlers
func (h handlers) secretAdd(obj interface{}) {
	sec := obj.(*v1.Secret)
	secKey := utils.GetResourceKey(sec.Namespace, sec.Name)
	if h.context.ingressSecretsMap.ContainsValue(secKey) {
		// find if this secKey exists in the map[string]UnorderedSets
		if err := h.context.CertificateSecretStore.convertSecret(secKey, sec); err == nil {
			currentTime := time.Now().UnixNano()
			h.context.LastSync = to.Int64Ptr(currentTime)
			h.context.Work <- events.Event{
				Type:      events.Create,
				Value:     obj,
				Timestamp: currentTime,
			}
		}
	}
}

func (h handlers) secretUpdate(oldObj, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}

	sec := newObj.(*v1.Secret)
	secKey := utils.GetResourceKey(sec.Namespace, sec.Name)
	if h.context.ingressSecretsMap.ContainsValue(secKey) {
		if err := h.context.CertificateSecretStore.convertSecret(secKey, sec); err == nil {
			currentTime := time.Now().UnixNano()
			h.context.LastSync = to.Int64Ptr(currentTime)
			h.context.Work <- events.Event{
				Type:      events.Update,
				Value:     newObj,
				Timestamp: currentTime,
			}
		}
	}
}

func (h handlers) secretDelete(obj interface{}) {
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
	h.context.CertificateSecretStore.delete(secKey)
	if h.context.ingressSecretsMap.ContainsValue(secKey) {
		currentTime := time.Now().UnixNano()
		h.context.LastSync = to.Int64Ptr(currentTime)
		h.context.Work <- events.Event{
			Type:      events.Delete,
			Value:     obj,
			Timestamp: currentTime,
		}
	}
}
