// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

// EventType represents the type of update for k8s resources
type EventType int

const (
	// Create type = 1
	Create EventType = iota + 1
	// Update type = 2
	Update
	// Delete type = 3
	Delete
)

// Event type that contains the type of event and data related to this event
type Event struct {
	Type  EventType
	Value interface{}
}
