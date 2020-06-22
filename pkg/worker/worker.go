// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package worker

import (
	"time"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

const sleepOnErrorSeconds = 5
const minTimeBetweenUpdates = 1 * time.Second

func drainChan(ch chan events.Event, epch chan events.Event, defaultEvent events.Event) events.Event {
	lastEvent := defaultEvent
	glog.V(9).Infof("Draining %d events from work channel", len(ch))
	for {
		select {
		case event := <-ch:
			// if there are more event in the queue
			// we will skip the reconcile event as we should focus on k8s related events
			if event.Type != events.PeriodicReconcile {
				// buffering endpoint event for later processing
				if _, endPointEvent := event.Value.(*v1.Endpoints); endPointEvent {
					epch <- event
				} else {
					lastEvent = event
				}
			}

		default:
			// if the last event is not endpoint type but the first event is, then buffer the first endpoint event for later batch
			// it garantees at least one endpoint event will be processed for mutation
			if _, endPointEvent := lastEvent.Value.(*v1.Endpoints); !endPointEvent {
				if _, endPointEvent := defaultEvent.Value.(*v1.Endpoints); endPointEvent {
					epch <- defaultEvent
				}
			}
			glog.V(5).Infof("Buffered %d endpoint events", len(epch))
			return lastEvent
		}
	}
}

// Run starts the worker which listens for events in eventChannel; stops when stopChannel is closed.
func (w *Worker) Run(eventChan chan events.Event, stopChannel chan struct{}) {
	endPointEventChan := make(chan events.Event, 1024)
	lastUpdate := time.Now().Add(-1 * time.Second)
	glog.V(1).Infoln("Worker started")
	for {
		select {
		case event := <-eventChan:
			if shouldProcess, reason := w.ShouldProcess(event); !shouldProcess {
				if reason != nil {
					// This log statement could potentially generate a large amount of log lines and most could be
					// innocuous - for instance: "endpoint default/aad-pod-identity-mic is not used by any Ingress"
					glog.V(9).Infof("Skipping event. Reason: %s", *reason)
				}
				continue
			}

			since := time.Since(lastUpdate)
			if since < minTimeBetweenUpdates {
				sleep := minTimeBetweenUpdates - since
				glog.V(9).Infof("[worker] It has been %+v since last update; Sleeping for %+v before next update", since, sleep)
				time.Sleep(sleep)
			}

			event = drainChan(eventChan, endPointEventChan, event)
			if err := w.ProcessEvent(event); err != nil {
				glog.Error("Error processing event.", err)
				time.Sleep(sleepOnErrorSeconds * time.Second)
			}

			lastUpdate = time.Now()
		case <-stopChannel:
			break
		default:
			// push one valid endpoint event back to work channel
			// note that at the same time, eventChan is still active to get more events.
			for len(endPointEventChan) > 0 {
				epEvent := <-endPointEventChan
				if shouldProcess, _ := w.ShouldProcess(epEvent); !shouldProcess {
					continue
				}
				glog.V(9).Info("###### Push back one endpoint event to the event channel ######")
				eventChan <- epEvent
				glog.V(9).Infof("###### %d events found after endpoint event push-back ######", len(eventChan))
				break
			}

			// we need only one valid endpoint event to trigger BackendAddressPools update, drain others if exist
			for len(endPointEventChan) > 0 {
				<-endPointEventChan
			}
		}
	}
}
