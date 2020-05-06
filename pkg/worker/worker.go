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

func drainChan(ch chan events.Event, defaultEvent events.Event) events.Event {
	lastEvent := defaultEvent

	glog.V(9).Infof("Draining %d events from work channel", len(ch))
	for {
		select {
		case event := <-ch:
			// if there are more event in the queue
			// we will skip the reconcile event as we should focus on k8s related events
			if event.Type != events.PeriodicReconcile {
				lastEvent = event
			}

			if _, endPointEvent := event.Value.(*v1.Endpoints); endPointEvent {
				glog.V(9).Info("############### endpoint event detected ###############")
				// stop drainning after feeding endpoint event back to the buffered channel
				// feeding back here is meant to not lose any endpoint events, e.g. endpoints events coming right after pod events
				// side effect is to have extra polls from appgw
				ch <- event
				return lastEvent
			}
		default:
			return lastEvent
		}
	}
}

func feedEndpointEvent(ch chan events.Event, defaultEvent events.Event) events.Event {
	lastEvent := defaultEvent
	glog.V(9).Infof("Draining %d events from work channel", len(ch))
	for {
		select {
		case event := <-ch:
			// if there are more event in the queue
			// we will skip the reconcile event as we should focus on k8s related events
			if event.Type != events.PeriodicReconcile {
				lastEvent = event
			}

			if _, yes := event.Value.(*v1.Endpoints); yes {
				ch <- event
				return lastEvent
			}
		default:
			return lastEvent
		}
	}
}

// Run starts the worker which listens for events in eventChannel; stops when stopChannel is closed.
func (w *Worker) Run(work chan events.Event, stopChannel chan struct{}) {
	lastUpdate := time.Now().Add(-1 * time.Second)
	glog.V(1).Infoln("Worker started")
	for {
		select {
		case event := <-work:
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

			_ = drainChan(work, event)

			if err := w.ProcessEvent(event); err != nil {
				glog.Error("Error processing event.", err)
				time.Sleep(sleepOnErrorSeconds * time.Second)
			}

			lastUpdate = time.Now()
		case <-stopChannel:
			break
		}
	}
}
