// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package worker

import (
	"encoding/json"
	"time"

	"github.com/eapache/channels"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

const sleepOnErrorSeconds = 5

// NewWorker creates a worker with a callback function. The callback
// function is executed for each event in the queue.
func NewWorker(processor EventProcessor) *Worker {
	w := &Worker{
		EventProcessor: processor,
	}

	return w
}

// Run starts the worker which listens for events in eventChannel. It loops until
// stopChannel is closed.
func (w *Worker) Run(eventChannel *channels.RingChannel, stopChannel chan struct{}) {
	go func() {
		for {
			select {
			case in := <-eventChannel.Out():
				event := in.(events.Event)

				if jsonEvent, err := json.Marshal(event); err != nil {
					glog.Error("Failed marshalling event:", err)
				} else {
					glog.V(5).Infof("Received event: %s", jsonEvent)
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
	}()
}
