package tests

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MockEventRecorder mocks EventRecorder, which knows how to record events on behalf of an EventSource.
type MockEventRecorder struct {}

// Event constructs an event from the given information and puts it in the queue for sending.
func (e MockEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {}

// Eventf is just like Event, but with Sprintf for the message field.
func (e MockEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {}

// PastEventf is just like Eventf, but with an option to specify the event's 'timestamp' field.
func (e MockEventRecorder) PastEventf(object runtime.Object, timestamp v1.Time, eventtype, reason, messageFmt string, args ...interface{}) {}

// AnnotatedEventf is just like eventf, but with annotations attached
func (e MockEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {}