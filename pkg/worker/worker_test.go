// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package worker

import (
	"time"

	"github.com/eapache/channels"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

var _ = Describe("Worker Test", func() {
	var stopChannel chan struct{}
	var eventChannel *channels.RingChannel

	BeforeEach(func() {
		stopChannel = make(chan struct{})
		eventChannel = channels.NewRingChannel(1024)
	})

	AfterEach(func() {
		close(stopChannel)
	})

	Context("Check that worker executes the process", func() {
		It("Should be able to run process func", func() {
			backChannel := make(chan struct{})
			eventProcessor := NewFakeProcessor(func(events.Event) error {
				backChannel <- struct{}{}
				return nil
			})
			worker := NewWorker(eventProcessor)
			worker.Run(eventChannel, stopChannel)

			ingress := *tests.NewIngressFixture()
			eventChannel.In() <- events.Event{
				Type:  events.Create,
				Value: ingress,
			}

			processCalled := false
			select {
			case <-backChannel:
				processCalled = true
				break
			case <-time.After(1 * time.Second):
				processCalled = false
			}

			Expect(processCalled).To(Equal(true), "Worker was not able to call process function within timeout")
		})
	})
})
