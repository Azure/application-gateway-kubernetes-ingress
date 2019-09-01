package k8scontext

import (
	"reflect"
	"time"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

type handlers struct {
	context *Context
}

// general resource handlers
func (h handlers) addFunc(obj interface{}) {
	currentTime := time.Now().UnixNano()
	h.context.LastSync = to.Int64Ptr(currentTime)
	h.context.Work <- events.Event{
		Type:      events.Create,
		Value:     obj,
		Timestamp: currentTime,
	}
}

func (h handlers) updateFunc(oldObj, newObj interface{}) {
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	currentTime := time.Now().UnixNano()
	h.context.LastSync = to.Int64Ptr(currentTime)
	h.context.Work <- events.Event{
		Type:      events.Update,
		Value:     newObj,
		Timestamp: currentTime,
	}
}

func (h handlers) deleteFunc(obj interface{}) {
	currentTime := time.Now().UnixNano()
	h.context.LastSync = to.Int64Ptr(currentTime)
	h.context.Work <- events.Event{
		Type:      events.Delete,
		Value:     obj,
		Timestamp: currentTime,
	}
}
