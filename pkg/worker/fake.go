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
	mutateAppGwy func() error
	mutateAKS    func([]events.Event) error
}

// MutateAppGateway will call the callback provided
func (fp FakeProcessor) MutateAppGateway() error {
	return fp.mutateAppGwy()
}

// MutateAKS will call the callback provided
func (fp FakeProcessor) MutateAKS(events []events.Event) error {
	return fp.mutateAKS(events)
}

// ShouldProcess will return true
func (fp FakeProcessor) ShouldProcess(event events.Event) (bool, *string) {
	return true, nil
}

// NewFakeProcessor returns a fake processor struct.
func NewFakeProcessor(mutateAppGwy func() error, mutateAKS func([]events.Event) error) FakeProcessor {
	return FakeProcessor{
		mutateAppGwy: mutateAppGwy,
		mutateAKS:    mutateAKS,
	}
}
