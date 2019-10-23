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
	MutateAppGateway(events.Event) error
	MutateAKS(events.Event) error
	ShouldProcess(events.Event) (bool, *string)
}

// Worker listens to the eventChannel and runs the EventProcessor.MutateAppGateway and MutateAKS
// for each event.
type Worker struct {
	EventProcessor
}
