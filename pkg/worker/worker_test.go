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

	Context("Check that worker executes the process", func() {
		It("Should be able to run process func", func() {
			backChannel := make(chan struct{})
			eventProcessor := NewFakeProcessor(func(events.Event) error {
				backChannel <- struct{}{}
				return nil
			})
			worker := Worker{
				EventProcessor: eventProcessor,
			}
			var lastSync *int64

			go worker.Run(work, lastSync, stopChannel)

			ingress := *tests.NewIngressFixture()
			work <- events.Event{
				Type:      events.Create,
				Value:     ingress,
				Timestamp: time.Now().Add(999 * time.Hour).UnixNano(),
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
				case work <- events.Event{Timestamp: counter}:
					counter++
				default:
					break Fill
				}
			}
			def := events.Event{Timestamp: int64(1234567890)}

			lastEvent := drainChan(work, def)

			Expect(lastEvent).To(Equal(events.Event{Timestamp: int64(9)}))
		})
	})

	Context("Verify that drainChan works", func() {
		It("Should drain the channel and return the default element", func() {
			buffSize := 10

			// Keep the channel empty
			work := make(chan events.Event, buffSize)
			def := events.Event{Timestamp: int64(1234567890)}
			lastEvent := drainChan(work, def)
			Expect(lastEvent).To(Equal(def))
		})
	})
})
