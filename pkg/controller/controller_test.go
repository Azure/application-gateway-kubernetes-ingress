// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	istioFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/worker"
)

var _ = Describe("test NewAppGwIngressController", func() {

	Context("ensure NewAppGwIngressController works as expected", func() {
		azClient := azure.NewFakeAzClient()
		appGwIdentifier := appgw.Identifier{}
		metricStore := metricstore.NewFakeMetricStore()
		k8sContext := &k8scontext.Context{MetricStore: metricStore}
		recorder := record.NewFakeRecorder(0)
		controller := NewAppGwIngressController(azClient, appGwIdentifier, k8sContext, recorder, metricStore, nil, false)
		It("should have created the AppGwIngressController struct", func() {
			err := controller.Start(environment.GetEnv())
			Expect(err).To(HaveOccurred())
			controller.Stop()
		})
	})

	Context("Verify that reconcilerTickerTask works", func() {
		var controller *AppGwIngressController

		BeforeEach(func() {
			// Create the mock K8s client.
			k8sClient := testclient.NewSimpleClientset()
			crdClient := fake.NewSimpleClientset()
			istioCrdClient := istioFake.NewSimpleClientset()
			// Create a `k8scontext` to start listening to ingress resources.
			k8scontext.IsNetworkingV1Beta1PackageSupported = true
			k8sContext := k8scontext.NewContext(k8sClient, crdClient, istioCrdClient, []string{}, 1000*time.Second, metricstore.NewFakeMetricStore())

			azClient := azure.NewFakeAzClient()
			appGwIdentifier := appgw.Identifier{}
			recorder := record.NewFakeRecorder(0)
			controller = NewAppGwIngressController(azClient, appGwIdentifier, k8sContext, recorder, metricstore.NewFakeMetricStore(), nil, false)
		})

		AfterEach(func() {
			close(controller.stopChannel)
		})

		It("should run reconcilePeriodSecondsStr if timeout is provided", func() {
			// create a fake worker which intercept the call and sends an event if reconcile event is received
			backChannel := make(chan struct{})
			processEvent := func(event events.Event) error {
				if event.Type == events.PeriodicReconcile {
					backChannel <- struct{}{}
				}
				return nil
			}
			eventProcessor := worker.NewFakeProcessor(processEvent)
			controller.worker = &worker.Worker{
				EventProcessor: eventProcessor,
			}

			env := environment.GetFakeEnv()
			env.ReconcilePeriodSeconds = "1"
			err := controller.Start(env)
			Expect(err).To(BeNil())

			// wait for reconcilerTickerTask to be added to the work queue
			processCalled := false
			select {
			case <-backChannel:
				processCalled = true
				break
			case <-time.After(3 * time.Second):
				processCalled = false
			}

			Expect(processCalled).To(Equal(true), "Reconciler didn't tick in the expected time.")
		})
	})
})
