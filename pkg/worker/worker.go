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

// Run starts the worker which listens for events in eventChannel; stops when stopChannel is closed.
func (w *Worker) Run(work chan events.Event, lastSync *int64, stopChannel chan struct{}) {
	for {
		select {
		case event := <-work:
			if shouldProcess, reason := w.ShouldProcess(event); !shouldProcess {
				if reason != "" {
					glog.V(5).Infof("Skipping event: %s", reason)
				}
				continue
			}

			if lastSync != nil && event.Timestamp < *lastSync {
				glog.V(5).Infof("Skipping event %d as time stamp is before last sync %d", event.Timestamp, *lastSync)
				continue
			}

			// Use callback to process event.
			if err := w.Process(event); err != nil {
				glog.Error("Processing event failed:", err)
				time.Sleep(sleepOnErrorSeconds * time.Second)
			} else {
				glog.V(3).Infoln("Successfully processed event")
			}
		case <-stopChannel:
			break
		}
	}
}
