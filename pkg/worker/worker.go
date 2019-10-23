// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package worker

import (
	"time"

	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

const sleepOnErrorSeconds = 5

func drainChan(ch chan events.Event, defaultEvent events.Event) (events.Event, []events.Event) {
	lastEvent := defaultEvent
	var allEvents []events.Event
	glog.V(9).Infof("Draining %d events from work channel", len(ch))
	for {
		select {
		case event := <-ch:
			allEvents = append(allEvents, event)
			lastEvent = event
		default:
			allEvents = append(allEvents, lastEvent)
			return lastEvent, allEvents
		}
	}
}

// Run starts the worker which listens for events in eventChannel; stops when stopChannel is closed.
func (w *Worker) Run(work chan events.Event, stopChannel chan struct{}) {
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

			_, allEvents := drainChan(work, event)

			if err := w.MutateAKS(allEvents); err != nil {
				glog.Error("Error mutating AKS from k8s event. ", err)
			}

			if err := w.MutateAppGateway(); err != nil {
				glog.Error("Error mutating App Gateway config from k8s event. ", err)
				time.Sleep(sleepOnErrorSeconds * time.Second)
			}

		case <-stopChannel:
			break
		}
	}
}
