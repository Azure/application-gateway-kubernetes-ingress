// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
)

// AppGwIngressController configures the application gateway based on the ingress rules defined.
type AppGwIngressController struct {
	appGwClient     network.ApplicationGatewaysClient
	appGwIdentifier appgw.Identifier

	k8sContext       *k8scontext.Context
	k8sUpdateChannel *channels.RingChannel

	eventQueue *EventQueue

	stopChannel chan struct{}
}

// NewAppGwIngressController constructs a controller object.
func NewAppGwIngressController(kubeclient kubernetes.Interface, appGwClient network.ApplicationGatewaysClient, appGwIdentifier appgw.Identifier, k8sContext *k8scontext.Context) *AppGwIngressController {
	controller := &AppGwIngressController{
		appGwIdentifier:  appGwIdentifier,
		k8sContext:       k8sContext,
		k8sUpdateChannel: k8sContext.UpdateChannel,
	}
	controller.eventQueue = NewEventQueue(controller.processEvent)
	controller.appGwClient = appGwClient
	return controller
}

// processEvent is the callback function that will be executed for every event
// in the EventQueue.
func (c *AppGwIngressController) processEvent(eventQueueElementInterface interface{}) (bool, error) {
	event := eventQueueElementInterface.(eventQueueElement)
	glog.V(1).Infof("controller.processEvent called with type %T", event.Element)

	ctx := context.Background()

	// Get current application gateway config
	appGw, err := c.appGwClient.Get(ctx, c.appGwIdentifier.ResourceGroup, c.appGwIdentifier.AppGwName)
	if err != nil {
		glog.Errorf("unable to get specified ApplicationGateway [%v], check ApplicationGateway identifier, error=[%v]", c.appGwIdentifier.AppGwName, err.Error())
		return false, errors.New("unable to get specified ApplicationGateway")
	}

	// Create a configbuilder based on current appgw config
	configBuilder := appgw.NewConfigBuilder(c.k8sContext, &c.appGwIdentifier, appGw.ApplicationGatewayPropertiesFormat)

	// Get all the ingresses
	ingressList := c.k8sContext.GetHTTPIngressList()

	// The following operations need to be in sequence
	configBuilder, err = configBuilder.BackendHTTPSettingsCollection(ingressList)
	if err != nil {
		glog.Errorf("unable to generate backend http settings, error [%v]", err.Error())
		return false, errors.New("unable to generate backend http settings")
	}

	// BackendAddressPools depend on BackendHTTPSettings
	configBuilder, err = configBuilder.BackendAddressPools(ingressList)
	if err != nil {
		glog.Errorf("unable to generate backend address pools, error [%v]", err.Error())
		return false, errors.New("unable to generate backend address pools")
	}

	// HTTPListener configures the frontend listeners
	configBuilder, err = configBuilder.HTTPListeners(ingressList)
	if err != nil {
		glog.Errorf("unable to generate frontend listeners, error [%v]", err.Error())
		return false, errors.New("unable to generate frontend listeners")
	}

	// RequestRoutingRules depends on the previous operations
	configBuilder, err = configBuilder.RequestRoutingRules(ingressList)
	if err != nil {
		glog.Errorf("unable to generate request routing rules, error [%v]", err.Error())
		return false, errors.New("unable to generate request routing rules")
	}

	// Replace the current appgw config with the generated one
	appGw.ApplicationGatewayPropertiesFormat = configBuilder.Build()

	glog.V(1).Info("~~~~~~~~ ↓ ApplicationGateway deployment ↓ ~~~~~~~~")
	defer glog.V(1).Info("~~~~~~~~ ↑ ApplicationGateway deployment ↑ ~~~~~~~~")

	deploymentStart := time.Now()
	// Initiate deployment
	appGwFuture, err := c.appGwClient.CreateOrUpdate(ctx, c.appGwIdentifier.ResourceGroup, c.appGwIdentifier.AppGwName, appGw)
	if err != nil {
		glog.Warningf("unable to send CreateOrUpdate request, error [%v]", err.Error())
		return false, errors.New("unable to send CreateOrUpdate request")
	}

	// Wait until deployment finshes and save the error message
	err = appGwFuture.WaitForCompletionRef(ctx, c.appGwClient.BaseClient.Client)
	deploymentElapsed := time.Now().Sub(deploymentStart)
	glog.V(1).Infof("deployment took %v", deploymentElapsed.String())

	if err != nil {
		glog.Warningf("unable to deploy ApplicationGateway, error [%v]", err.Error())
		return false, errors.New("unable to deploy ApplicationGateway")
	}

	return true, nil
}

// Start function runs the k8scontext and continues to listen to the
// event channel and enqueue events before stopChannel is closed
func (c *AppGwIngressController) Start() {
	// Starts event queue
	go c.eventQueue.Run(time.Second, c.stopChannel)

	// Starts k8scontext which contains all the informers
	c.k8sContext.Run()

	// Continue to enqueue events into eventqueue until stopChannel is closed
	for {
		select {
		case obj := <-c.k8sUpdateChannel.Out():
			event := obj.(k8scontext.Event)
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
