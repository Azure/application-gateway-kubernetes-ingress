// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package events

// EventType is the type of event we have received from Kubernetes
type EventType int

const (
	// Create type = 1
	Create EventType = iota + 1
	// Update type = 2
	Update
	// Delete type = 3
	Delete
)

// Event is the combined type and actual object we received from Kubernetes
type Event struct {
	Type  EventType
	Value interface{}
}
