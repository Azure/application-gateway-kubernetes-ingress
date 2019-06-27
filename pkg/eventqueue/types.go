// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package eventqueue

import (
	"k8s.io/client-go/util/workqueue"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

// EventProcessor provides a mechanism to act on events in the internal queue.
type EventProcessor interface {
	Process(QueuedEvent) error
}

// EventQueue is a queue accepting events and run callback function
// for each events.
type EventQueue struct {
	EventProcessor

	queue              workqueue.RateLimitingInterface
	workerFinished     chan struct{}
	lastEventTimestamp int64
}

// QueuedEvent encapsulates an event with timestamp and a canSkip
// configuration. CanSkip specifies if this event can be skipped if a previous
// event is processed at a later time.
type QueuedEvent struct {
	Event     events.Event
	Timestamp int64
	CanSkip   bool
}
