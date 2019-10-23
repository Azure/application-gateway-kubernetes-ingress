// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package worker

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

// FakeProcessor is fake event processor type
type FakeProcessor struct {
	processFunc func(events.Event) error
}

// MutateAppGateway will call the callback provided
func (fp FakeProcessor) MutateAppGateway(event events.Event) error {
	return fp.processFunc(event)
}

// MutateAKS will call the callback provided
func (fp FakeProcessor) MutateAKS(event events.Event) error {
	return fp.processFunc(event)
}

// ShouldProcess will return true
func (fp FakeProcessor) ShouldProcess(event events.Event) (bool, *string) {
	return true, nil
}

// NewFakeProcessor returns a fake processor struct.
func NewFakeProcessor(process func(events.Event) error) FakeProcessor {
	return FakeProcessor{
		processFunc: process,
	}
}
