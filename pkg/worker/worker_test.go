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
})
