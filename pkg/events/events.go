// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package events

// EventType is the type of event we have received from Kubernetes
type EventType int

const (
	// Create is a type of a Kubernetes API event.
	Create EventType = iota + 1

	// Update is a type of a Kubernetes API event.
	Update

	// Delete is a type of a Kubernetes API event.
	Delete

	// PeriodicReconcile is a event for periodically reconciling Application Gateway
	PeriodicReconcile
)

// Event is the combined type and actual object we received from Kubernetes
type Event struct {
	Type  EventType
	Value interface{}
}
