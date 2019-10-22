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
	h.context.Work <- events.Event{
		Type:  events.Create,
		Value: obj,
	}
	h.context.metricStore.IncK8sAPIEventCounter()
}

func (h handlers) updateFunc(oldObj, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	h.context.Work <- events.Event{
		Type:  events.Update,
		Value: newObj,
	}
	h.context.metricStore.IncK8sAPIEventCounter()
}

func (h handlers) deleteFunc(obj interface{}) {
	h.context.Work <- events.Event{
		Type:  events.Delete,
		Value: obj,
	}
	h.context.metricStore.IncK8sAPIEventCounter()
}
