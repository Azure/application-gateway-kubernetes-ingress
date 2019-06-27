// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/eapache/channels"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/eventqueue"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

// AppGwIngressController configures the application gateway based on the ingress rules defined.
type AppGwIngressController struct {
	appGwClient     n.ApplicationGatewaysClient
	appGwIdentifier appgw.Identifier

	k8sContext       *k8scontext.Context
	k8sUpdateChannel *channels.RingChannel

	eventQueue *eventqueue.EventQueue

	configCache *[]byte
	stopChannel chan struct{}

	recorder record.EventRecorder
}

// NewAppGwIngressController constructs a controller object.
func NewAppGwIngressController(appGwClient n.ApplicationGatewaysClient, appGwIdentifier appgw.Identifier, k8sContext *k8scontext.Context, recorder record.EventRecorder) *AppGwIngressController {
	controller := &AppGwIngressController{
		appGwClient:      appGwClient,
		appGwIdentifier:  appGwIdentifier,
		k8sContext:       k8sContext,
		k8sUpdateChannel: k8sContext.UpdateChannel,
		recorder:         recorder,
		configCache:      to.ByteSlicePtr([]byte{}),
	}

	controller.eventQueue = eventqueue.NewEventQueue(controller)
	return controller
}

// Start function runs the k8scontext and continues to listen to the
// event channel and enqueue events before stopChannel is closed
func (c *AppGwIngressController) Start(envVariables environment.EnvVariables) {
	// Starts event queue
	go c.eventQueue.Run(time.Second, c.stopChannel)

	// Starts k8scontext which contains all the informers
	c.k8sContext.Run(false, envVariables)

	// Continue to enqueue events into eventqueue until stopChannel is closed
	for {
		select {
		case obj := <-c.k8sUpdateChannel.Out():
			event := obj.(events.Event)
			c.eventQueue.Enqueue(event)
		case <-c.stopChannel:
			break
		}
	}
}

// Stop function terminates the k8scontext and signal the stopchannel
func (c *AppGwIngressController) Stop() {
	c.k8sContext.Stop()
	c.stopChannel <- struct{}{}
}
