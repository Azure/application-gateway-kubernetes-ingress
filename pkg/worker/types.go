// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package worker

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

// EventProcessor provides a mechanism to act on events in the internal queue.
type EventProcessor interface {
	Process(events.Event) error
}

// Worker listens on the eventChannel and runs the EventProcessor.Process
// for each event.
type Worker struct {
	EventProcessor
}
