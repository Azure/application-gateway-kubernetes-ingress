// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package worker

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

// FakeProcessor is fake event processor type
type FakeEventProcessor struct {
	processEventCallBack func(event events.Event) error
}

// ProcessEvent will invoke the callback provided
func (fp FakeEventProcessor) ProcessEvent(event events.Event) error {
	return fp.processEventCallBack(event)
}

// ShouldProcess will return true
func (fp FakeEventProcessor) ShouldProcess(event events.Event) (bool, *string) {
	return true, nil
}

// NewFakeProcessor returns a fake processor struct.
func NewFakeProcessor(processEvent func(event events.Event) error) FakeEventProcessor {
	return FakeEventProcessor{
		processEventCallBack: processEvent,
	}
}
