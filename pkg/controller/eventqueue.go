// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"time"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

// EventQueue is a queue accepting events and run callback function
// for each events.
type EventQueue struct {
	queue              workqueue.RateLimitingInterface
	process            func(interface{}) (bool, error)
	workerFinished     chan struct{}
	lastEventTimestamp int64
}

// eventQueueElement encapsulates an event with timestamp and a canSkip
// configuration. CanSkip specifies if this event can be skipped if a previous
// event is processed at a later time.
type eventQueueElement struct {
	Element   interface{}
	Timestamp int64
	CanSkip   bool
}

// NewEventQueue creates an EventQueue with a callback function. The callback
// function processFunc is executed for each event in the queue.
func NewEventQueue(processFunc func(interface{}) (bool, error)) *EventQueue {
	q := &EventQueue{
		queue:              workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		process:            processFunc,
		workerFinished:     make(chan struct{}),
		lastEventTimestamp: int64(0),
	}

	return q
}

// EnqueueCanSkip adds an event with parameter el as payload. User can specify if
// this event should be skippable by setting the boolean parameter skip.
func (q *EventQueue) EnqueueCanSkip(el interface{}, skip bool) {
	if q.queue.ShuttingDown() {
		// Queue is shutting down will not be able to enqueue this.
		glog.Errorf("queue is shutting down, unable to enqueue event")
		return
	}

	now := time.Now().UnixNano()

	glog.V(1).Infof("Enqueuing skip(%v) item", skip)

	v := eventQueueElement{
		Element:   el,
		Timestamp: now,
		CanSkip:   skip,
	}

	q.queue.Add(v)
}

// Enqueue adds an non-skipable event with parameter el as payload.
func (q *EventQueue) Enqueue(el interface{}) {
	q.EnqueueCanSkip(el, false)
}

// Shutdown closes the queue and waits until the last callback is finished.
// After shutdown, the EventQueue will not accept any events. Shutdown waits
// until callback finishes if a callback is processing an event.
func (q *EventQueue) Shutdown() {
	q.queue.ShutDown()
	<-q.workerFinished
}

// Run starts the queue's worker and restarts every period time. It loops until
// stopChannel is closed.
func (q *EventQueue) Run(period time.Duration, stopChannel chan struct{}) {
	wait.Until(q.worker, period, stopChannel)
}

// isChanClosed tests if a channel is closed without waiting the channel.
func isChanClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
	}
	return false
}

func (q *EventQueue) worker() {
	for {
		in, shutdown := q.queue.Get()
		if shutdown {
			// The event queue is shutting down.
			break
		}
		event := in.(eventQueueElement)

		now := time.Now().UnixNano()

		if event.CanSkip && (q.lastEventTimestamp > event.Timestamp) {
			// Skip this event
			glog.V(1).Infof("Skipping event")
			q.queue.Forget(event)
			q.queue.Done(event)
			continue
		}

		glog.V(1).Infof("Processing event begin, time since event generation: %s", time.Duration(now-event.Timestamp).String())

		// Use callback to process event.
		_, err := q.process(event)

		if err != nil {
			// TODO maybe we can implement retry logic for scenarios like failed network.
			glog.V(1).Infoln("Processing event failed")
		} else {
			glog.V(1).Infoln("Processing event done, updating lastEventTimestamp")
			q.queue.Forget(event)
			q.lastEventTimestamp = utils.MaxInt64(q.lastEventTimestamp, event.Timestamp)
		}

		q.queue.Done(event)
	}
	// Close channel.
	if isChanClosed(q.workerFinished) {
		close(q.workerFinished)
	}
}
