// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package worker

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

var _ = Describe("Worker Test", func() {
	var stopChannel chan struct{}
	var work chan events.Event

	BeforeEach(func() {
		stopChannel = make(chan struct{})
		work = make(chan events.Event)
	})

	AfterEach(func() {
		close(stopChannel)
	})

	Context("Check that worker processes the event", func() {
		It("Should be able to run process func", func() {
			backChannel := make(chan struct{})
			processEvent := func(event events.Event) error {
				backChannel <- struct{}{}
				return nil
			}
			eventProcessor := NewFakeProcessor(processEvent)
			worker := Worker{
				EventProcessor: eventProcessor,
			}
			go worker.Run(work, stopChannel)

			ingress := *tests.NewIngressFixture()
			work <- events.Event{
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

	Context("Verify that drainChan works", func() {
		It("Should drain the channel and return the last element", func() {
			buffSize := 10
			counter := int64(0)

			// Create and fill the channel
			work := make(chan events.Event, buffSize)
		Fill:
			for {
				select {
				case work <- events.Event{}:
					counter++
				default:
					break Fill
				}
			}
			Expect(counter).To(Equal(int64(len(work))))
			def := events.Event{}
			lastEvent := drainChan(work, def)
			Expect(len(work)).To(Equal(0))
			Expect(lastEvent).To(Equal(events.Event{}))
		})
	})

	Context("Verify that drainChan works", func() {
		It("Should drain the channel and return the default element", func() {
			buffSize := 10

			// Keep the channel empty
			work := make(chan events.Event, buffSize)
			def := events.Event{}
			lastEvent := drainChan(work, def)
			Expect(lastEvent).To(Equal(def))
		})
	})
})