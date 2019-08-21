// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/worker"
)

// AppGwIngressController configures the application gateway based on the ingress rules defined.
type AppGwIngressController struct {
	appGwClient     n.ApplicationGatewaysClient
	appGwIdentifier appgw.Identifier
	ipAddressMap    map[string]k8scontext.IPAddress

	k8sContext *k8scontext.Context
	worker     *worker.Worker

	configCache *[]byte

	recorder record.EventRecorder

	stopChannel chan struct{}
}

// NewAppGwIngressController constructs a controller object.
func NewAppGwIngressController(appGwClient n.ApplicationGatewaysClient, appGwIdentifier appgw.Identifier, k8sContext *k8scontext.Context, recorder record.EventRecorder) *AppGwIngressController {
	controller := &AppGwIngressController{
		appGwClient:     appGwClient,
		appGwIdentifier: appGwIdentifier,
		k8sContext:      k8sContext,
		recorder:        recorder,
		configCache:     to.ByteSlicePtr([]byte{}),
		ipAddressMap:    map[string]k8scontext.IPAddress{},
		stopChannel:     make(chan struct{}),
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
	go c.worker.Run(c.k8sContext.UpdateChannel, c.stopChannel)
	return nil
}

// Stop function terminates the k8scontext and signal the stopchannel
func (c *AppGwIngressController) Stop() {
	close(c.stopChannel)
}

// Liveness fulfills the health.HealthProbe interface; It is evaluated when K8s liveness-checks the AGIC pod.
func (c *AppGwIngressController) Liveness() bool {
	// TODO(draychev): implement
	return true
}

// Readiness fulfills the health.HealthProbe interface; It is evaluated when K8s readiness-checks the AGIC pod.
func (c *AppGwIngressController) Readiness() bool {
	_, isOpen := <-c.k8sContext.CacheSynced
	// When the channel is CLOSED we have synced cache and are READY!
	return !isOpen
}
