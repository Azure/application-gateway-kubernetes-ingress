package k8scontext

import (
	"reflect"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type handlers struct {
	context *Context
}

// general resource handlers
func (h handlers) addFunc(obj interface{}) {
	ns := getNamespace(obj)
	if _, exists := namespacesToIgnore[ns]; exists {
		return
	}
	if _, exists := h.context.namespaces[ns]; len(h.context.namespaces) > 0 && !exists {
		return
	}

	h.context.Work <- events.Event{
		Type:  events.Create,
		Value: obj,
	}
	h.context.MetricStore.IncK8sAPIEventCounter()
}

func (h handlers) updateFunc(oldObj, newObj interface{}) {
	ns := getNamespace(newObj)
	if _, exists := namespacesToIgnore[ns]; exists {
		return
	}
	if _, exists := h.context.namespaces[ns]; len(h.context.namespaces) > 0 && !exists {
		return
	}

	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	h.context.Work <- events.Event{
		Type:  events.Update,
		Value: newObj,
	}
	h.context.MetricStore.IncK8sAPIEventCounter()
}

func (h handlers) deleteFunc(obj interface{}) {
	obj, ok := unwrapTombstone(obj)
	if !ok {
		return
	}

	ns := getNamespace(obj)
	if _, exists := namespacesToIgnore[ns]; exists {
		return
	}
	if _, exists := h.context.namespaces[ns]; len(h.context.namespaces) > 0 && !exists {
		return
	}

	h.context.Work <- events.Event{
		Type:  events.Delete,
		Value: obj,
	}
	h.context.MetricStore.IncK8sAPIEventCounter()
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

// unwrapTombstone returns the object a cache.DeletedFinalStateUnknown tombstone wraps, or
// obj unchanged if it isn't one. ok is false if the tombstone has no recoverable object,
// in which case the caller should drop the event.
func unwrapTombstone(obj interface{}) (interface{}, bool) {
	tombstone, isTombstone := obj.(cache.DeletedFinalStateUnknown)
	if !isTombstone {
		return obj, true
	}
	if tombstone.Obj == nil {
		klog.Errorf("unable to get object from tombstone with key %s", tombstone.Key)
		return nil, false
	}
	return tombstone.Obj, true
}
