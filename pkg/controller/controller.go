// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/eapache/channels"
	"github.com/golang/glog"
)

// AppGwIngressController configures the application gateway based on the ingress rules defined.
type AppGwIngressController struct {
	appGwClient     network.ApplicationGatewaysClient
	appGwIdentifier appgw.Identifier

	k8sContext       *k8scontext.Context
	k8sUpdateChannel *channels.RingChannel

	eventQueue *EventQueue

	configCache *[]byte
	stopChannel chan struct{}

	recorder record.EventRecorder
}

// NewAppGwIngressController constructs a controller object.
func NewAppGwIngressController(appGwClient network.ApplicationGatewaysClient, appGwIdentifier appgw.Identifier, k8sContext *k8scontext.Context, recorder record.EventRecorder) *AppGwIngressController {
	controller := &AppGwIngressController{
		appGwClient:      appGwClient,
		appGwIdentifier:  appGwIdentifier,
		k8sContext:       k8sContext,
		k8sUpdateChannel: k8sContext.UpdateChannel,
		recorder:         recorder,
	}

	controller.eventQueue = NewEventQueue(controller)
	return controller
}

// Process is the callback function that will be executed for every event
// in the EventQueue.
func (c AppGwIngressController) Process(event QueuedEvent) error {
	glog.V(1).Infof("controller.Process called with type %T", event.Event)

	ctx := context.Background()

	// Get current application gateway config
	appGw, err := c.appGwClient.Get(ctx, c.appGwIdentifier.ResourceGroup, c.appGwIdentifier.AppGwName)
	if err != nil {
		glog.Errorf("unable to get specified ApplicationGateway [%v], check ApplicationGateway identifier, error=[%v]", c.appGwIdentifier.AppGwName, err.Error())
		return errors.New("unable to get specified ApplicationGateway")
	}

	// Create a configbuilder based on current appgw config
	configBuilder := appgw.NewConfigBuilder(c.k8sContext, &c.appGwIdentifier, appGw.ApplicationGatewayPropertiesFormat, c.recorder)

	// Get all the ingresses and services
	ingressList := c.k8sContext.GetHTTPIngressList()
	serviceList := c.k8sContext.GetServiceList()

	// The following operations need to be in sequence
	err = configBuilder.HealthProbesCollection(ingressList, serviceList)
	if err != nil {
		glog.Errorf("unable to generate Health Probes, error [%v]", err.Error())
		return errors.New("unable to generate health probes")
	}

	// The following operations need to be in sequence
	err = configBuilder.BackendHTTPSettingsCollection(ingressList, serviceList)
	if err != nil {
		glog.Errorf("unable to generate backend http settings, error [%v]", err.Error())
		return errors.New("unable to generate backend http settings")
	}

	// BackendAddressPools depend on BackendHTTPSettings
	err = configBuilder.BackendAddressPools(ingressList, serviceList)
	if err != nil {
		glog.Errorf("unable to generate backend address pools, error [%v]", err.Error())
		return errors.New("unable to generate backend address pools")
	}

	// HTTPListener configures the frontend listeners
	// This also creates redirection configuration (if TLS is configured and Ingress is annotated).
	// This configuration must be attached to request routing rules, which are created in the steps below.
	// The order of operations matters.
	err = configBuilder.Listeners(ingressList)
	if err != nil {
		glog.Errorf("unable to generate frontend listeners, error [%v]", err.Error())
		return errors.New("unable to generate frontend listeners")
	}

	// SSL redirection configurations created elsewhere will be attached to the appropriate rule in this step.
	err = configBuilder.RequestRoutingRules(ingressList, serviceList)
	if err != nil {
		glog.Errorf("unable to generate request routing rules, error [%v]", err.Error())
		return errors.New("unable to generate request routing rules")
	}

	// Replace the current appgw config with the generated one
	if appGw.ApplicationGatewayPropertiesFormat, err = configBuilder.Build(); err != nil {
		glog.Error("ConfigBuilder failed to create Application Gateway config:", err)
	}

	addTags(&appGw)

	if c.configIsSame(&appGw) {
		glog.Infoln("cache: Config has NOT changed! No need to connect to ARM.")
		return nil
	}

	glog.V(1).Info("BEGIN ApplicationGateway deployment")
	defer glog.V(1).Info("END ApplicationGateway deployment")

	deploymentStart := time.Now()
	// Initiate deployment
	appGwFuture, err := c.appGwClient.CreateOrUpdate(ctx, c.appGwIdentifier.ResourceGroup, c.appGwIdentifier.AppGwName, appGw)
	if err != nil {
		// Reset cache
		c.configCache = nil
		glog.Warningf("unable to send CreateOrUpdate request, error [%v]", err.Error())
		configJSON, _ := c.dumpSanitizedJSON(&appGw)
		glog.V(5).Info(string(configJSON))
		return errors.New("unable to send CreateOrUpdate request")
	}
	// Wait until deployment finshes and save the error message
	err = appGwFuture.WaitForCompletionRef(ctx, c.appGwClient.BaseClient.Client)
	configJSON, _ := c.dumpSanitizedJSON(&appGw)
	glog.V(5).Info(string(configJSON))
	glog.V(1).Infof("deployment took %+v", time.Now().Sub(deploymentStart).String())

	if err != nil {
		// Reset cache
		c.configCache = nil
		glog.Warningf("unable to deploy ApplicationGateway, error [%v]", err.Error())
		return errors.New("unable to deploy ApplicationGateway")
	}

	glog.Info("cache: Updated with latest applied config.")
	c.updateCache(&appGw)

	return nil
}

// addTags will add certain tags to Application Gateway
func addTags(appGw *network.ApplicationGateway) {
	if appGw.Tags == nil {
		appGw.Tags = make(map[string]*string)
	}
	// Identify the App Gateway as being exclusively managed by a Kubernetes Ingress.
	appGw.Tags[managedByK8sIngress] = to.StringPtr(fmt.Sprintf("%s/%s/%s", version.Version, version.GitCommit, version.BuildDate))
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
