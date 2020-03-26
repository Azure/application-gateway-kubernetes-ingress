// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/worker"
)

// AppGwIngressController configures the application gateway based on the ingress rules defined.
type AppGwIngressController struct {
	azClient        azure.AzClient
	appGwIdentifier appgw.Identifier
	ipAddressMap    map[string]k8scontext.IPAddress

	k8sContext       *k8scontext.Context
	worker           *worker.Worker
	hostedOnUnderlay bool

	configCache *[]byte

	recorder record.EventRecorder

	agicPod     *v1.Pod
	metricStore metricstore.MetricStore

	stopChannel chan struct{}
}

// NewAppGwIngressController constructs a controller object.
func NewAppGwIngressController(azClient azure.AzClient, appGwIdentifier appgw.Identifier, k8sContext *k8scontext.Context, recorder record.EventRecorder, metricStore metricstore.MetricStore, agicPod *v1.Pod, hostedOnUnderlay bool) *AppGwIngressController {
	controller := &AppGwIngressController{
		azClient:         azClient,
		appGwIdentifier:  appGwIdentifier,
		k8sContext:       k8sContext,
		recorder:         recorder,
		configCache:      to.ByteSlicePtr([]byte{}),
		ipAddressMap:     map[string]k8scontext.IPAddress{},
		stopChannel:      make(chan struct{}),
		agicPod:          agicPod,
		metricStore:      metricStore,
		hostedOnUnderlay: hostedOnUnderlay,
	}

	controller.worker = &worker.Worker{
		EventProcessor: controller,
	}
	return controller
}

// Start function runs the k8scontext and continues to listen to the
// event channel and enqueue events before stopChannel is closed
func (c *AppGwIngressController) Start(envVariables environment.EnvVariables) error {
	// Starts k8scontext which contains all the informers
	// This will start individual go routines for informers
	if err := c.k8sContext.Run(c.stopChannel, false, envVariables); err != nil {
		glog.Error("Could not start Kubernetes Context: ", err)
		return err
	}

	// Starts Worker processing events from k8sContext
	go c.worker.Run(c.k8sContext.Work, c.stopChannel, envVariables.ReconcilePeriodSeconds)
	return nil
}

// Stop function terminates the k8scontext and signal the stopchannel
func (c *AppGwIngressController) Stop() {
	c.metricStore.Stop()
	close(c.stopChannel)
}

// Liveness fulfills the health.HealthProbe interface; It is evaluated when K8s liveness-checks the AGIC pod.
func (c *AppGwIngressController) Liveness() bool {
	// TODO(draychev): implement
	return true
}

// Readiness fulfills the health.HealthProbe interface; It is evaluated when K8s readiness-checks the AGIC pod.
func (c *AppGwIngressController) Readiness() bool {
	if !c.hostedOnUnderlay {
		// When the channel is CLOSED we have synced cache and are READY!
		_, isOpen := <-c.k8sContext.CacheSynced
		return !isOpen
	}

	return true
}

// ProcessEvent is the handler for K8 cluster events which are listened by informers.
func (c *AppGwIngressController) ProcessEvent(event events.Event) error {
	appGw, cbCtx, err := c.GetAppGw()
	if err != nil {
		glog.Error("Error Retrieving AppGw for k8s event. ", err)
		return err
	}

	// Reset all ingress Ips and igore mutating appgw if gateway is in stopped state
	if !c.isApplicationGatewayMutable(appGw) {
		glog.Info("Reset all ingress ip")
		c.ResetAllIngress(appGw, cbCtx)
		glog.Info("Ignore mutating App Gateway as it is not mutable")
		return nil
	}

	if err := c.MutateAllIngress(appGw, cbCtx); err != nil {
		glog.Error("Error mutating AKS from k8s event. ", err)
	}

	invokedForReconciliation := event.Type == events.PeriodicReconcile
	glog.Info("[reconcile] triggered: ", invokedForReconciliation)
	if err := c.MutateAppGateway(appGw, cbCtx, invokedForReconciliation); err != nil {
		glog.Error("Error mutating App Gateway config from k8s event. ", err)
	}

	return nil
}
