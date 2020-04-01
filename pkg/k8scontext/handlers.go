package k8scontext

import (
	"reflect"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
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
