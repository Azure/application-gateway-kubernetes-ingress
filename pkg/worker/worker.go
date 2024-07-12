// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package worker

import (
	"reflect"
	"time"

	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

const sleepOnErrorSeconds = 5
const minTimeBetweenUpdates = 1 * time.Second

func drainChan(ch chan events.Event, defaultEvent events.Event) events.Event {
	lastEvent := defaultEvent
	klog.V(9).Infof("Draining %d events from work channel", len(ch))
	for {
		select {
		case event := <-ch:
			// if there are more event in the queue
			// we will skip the reconcile event as we should focus on k8s related events
			if event.Type != events.PeriodicReconcile {
				lastEvent = event
			}
		default:
			return lastEvent
		}
	}
}

// Run starts the worker which listens for events in eventChannel; stops when stopChannel is closed.
func (w *Worker) Run(work chan events.Event, stopChannel chan struct{}) {
	lastUpdate := time.Now().Add(-1 * time.Second)
	klog.V(1).Infoln("Worker started")
	for {
		select {
		case event := <-work:
			if shouldProcess, reason := w.ShouldProcess(event); !shouldProcess {
				if reason != nil {
					// This log statement could potentially generate a large amount of log lines and most could be
					// innocuous - for instance: "endpoint default/aad-pod-identity-mic is not used by any Ingress"
					klog.V(3).Infof("Skipping event. Reason: %s", *reason)
				}
				continue
			}

			// get name, namespace and kind from event.Value
			name := reflect.ValueOf(event.Value).Elem().FieldByName("Name").String()
			namespace := reflect.ValueOf(event.Value).Elem().FieldByName("Namespace").String()
			objectType := reflect.TypeOf(event.Value).Elem()

			klog.V(3).Infof("Processing k8s event of type:%s object:%s/%s/%s", event.Type, objectType, namespace, name)

			since := time.Since(lastUpdate)
			if since < minTimeBetweenUpdates {
				sleep := minTimeBetweenUpdates - since
				klog.V(9).Infof("[worker] It has been %+v since last update; Sleeping for %+v before next update", since, sleep)
				time.Sleep(sleep)
			}

			_ = drainChan(work, event)

			if err := w.ProcessEvent(event); err != nil {
				klog.Error("Error processing event.", err)
				time.Sleep(sleepOnErrorSeconds * time.Second)
			}

			lastUpdate = time.Now()
		case <-stopChannel:
			break
		}
	}
}
