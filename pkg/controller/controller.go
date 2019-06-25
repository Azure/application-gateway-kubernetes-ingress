// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
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
	ctx := context.Background()

	// Get current application gateway config
	appGw, err := c.appGwClient.Get(ctx, c.appGwIdentifier.ResourceGroup, c.appGwIdentifier.AppGwName)
	if err != nil {
		glog.Errorf("unable to get specified ApplicationGateway [%v], check ApplicationGateway identifier, error=[%v]", c.appGwIdentifier.AppGwName, err.Error())
		return errors.New("unable to get specified ApplicationGateway")
	}

	// Create a configbuilder based on current appgw config
	configBuilder := appgw.NewConfigBuilder(c.k8sContext, &c.appGwIdentifier, appGw.ApplicationGatewayPropertiesFormat, c.recorder)

	cbCtx := &appgw.ConfigBuilderContext{
		// Get all Services
		ServiceList:       c.k8sContext.GetServiceList(),
		IngressList:       c.k8sContext.GetHTTPIngressList(),
		ManagedTargets:    c.k8sContext.GetAzureIngressManagedTargets(),
		ProhibitedTargets: c.k8sContext.GetAzureProhibitedTargets(),
		EnvVariables:      environment.GetEnv(),
		IstioGateways:     c.k8sContext.GetIstioGateways(),
	}
	{
		var managedTargets []string
		for _, target := range cbCtx.ManagedTargets {
			managedTargets = append(managedTargets, fmt.Sprintf("%s/%s", target.Namespace, target.Name))
		}
		glog.V(5).Infof("AzureIngressManagedTargets: %+v", strings.Join(managedTargets, ","))
	}
	{
		var prohibitedTargets []string
		for _, target := range cbCtx.ProhibitedTargets {
			prohibitedTargets = append(prohibitedTargets, fmt.Sprintf("%s/%s", target.Namespace, target.Name))
		}

		glog.V(5).Infof("AzureIngressProhibitedTargets: %+v", strings.Join(prohibitedTargets, ","))
	}
	if cbCtx.EnvVariables.EnableIstioIntegration == "true" {
		var gatewaysInfo []string
		for _, gateway := range cbCtx.IstioGateways {
			gatewaysInfo = append(gatewaysInfo, fmt.Sprintf("%s/%s", gateway.Namespace, gateway.Name))
		}
		glog.V(5).Infof("Istio Gateways: %+v", strings.Join(gatewaysInfo, ","))
	}

	// Run fatal validations on the existing config of the Application Gateway.
	if err := appgw.FatalValidateOnExistingConfig(c.recorder, appGw.ApplicationGatewayPropertiesFormat, cbCtx.EnvVariables); err != nil {
		glog.Error("Got a fatal validation error on existing Application Gateway config. Will retry getting Application Gateway until error is resolved:", err)
		return err
	}

	// Run validations on the Kubernetes resources which can suggest misconfiguration.
	if err = configBuilder.PreBuildValidate(cbCtx); err != nil {
		glog.Error("ConfigBuilder PostBuildValidate returned error:", err)
	}

	// The following operations need to be in sequence
	err = configBuilder.HealthProbesCollection(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate Health Probes, error [%v]", err.Error())
		return errors.New("unable to generate health probes")
	}

	// The following operations need to be in sequence
	err = configBuilder.BackendHTTPSettingsCollection(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate backend http settings, error [%v]", err.Error())
		return errors.New("unable to generate backend http settings")
	}

	// BackendAddressPools depend on BackendHTTPSettings
	err = configBuilder.BackendAddressPools(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate backend address pools, error [%v]", err.Error())
		return errors.New("unable to generate backend address pools")
	}

	// HTTPListener configures the frontend listeners
	// This also creates redirection configuration (if TLS is configured and Ingress is annotated).
	// This configuration must be attached to request routing rules, which are created in the steps below.
	// The order of operations matters.
	err = configBuilder.Listeners(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate frontend listeners, error [%v]", err.Error())
		return errors.New("unable to generate frontend listeners")
	}

	// SSL redirection configurations created elsewhere will be attached to the appropriate rule in this step.
	err = configBuilder.RequestRoutingRules(cbCtx)
	if err != nil {
		glog.Errorf("unable to generate request routing rules, error [%v]", err.Error())
		return errors.New("unable to generate request routing rules")
	}

	// Run post validations to report errors in the config generation.
	if err = configBuilder.PostBuildValidate(cbCtx); err != nil {
		glog.Error("ConfigBuilder PostBuildValidate returned error:", err)
	}

	// Replace the current appgw config with the generated one
	appGw.ApplicationGatewayPropertiesFormat = configBuilder.GetApplicationGatewayPropertiesFormatPtr()

	addTags(&appGw)

	if c.configIsSame(&appGw) {
		glog.V(3).Info("cache: Config has NOT changed! No need to connect to ARM.")
		return nil
	}

	glog.V(3).Info("BEGIN ApplicationGateway deployment")
	defer glog.V(3).Info("END ApplicationGateway deployment")

	deploymentStart := time.Now()
	// Initiate deployment
	appGwFuture, err := c.appGwClient.CreateOrUpdate(ctx, c.appGwIdentifier.ResourceGroup, c.appGwIdentifier.AppGwName, appGw)
	if err != nil {
		// Reset cache
		c.configCache = nil
		configJSON, _ := c.dumpSanitizedJSON(&appGw)
		glog.Errorf("Failed applying App Gwy configuration: %s -- %s", err, string(configJSON))
		return err
	}
	// Wait until deployment finshes and save the error message
	err = appGwFuture.WaitForCompletionRef(ctx, c.appGwClient.BaseClient.Client)
	configJSON, _ := c.dumpSanitizedJSON(&appGw)
	glog.V(5).Info(string(configJSON))

	// We keep this at log level 1 to show some heartbeat in the logs. Without this it is way too quiet.
	glog.V(1).Infof("Applied App Gateway config in %+v", time.Now().Sub(deploymentStart).String())

	if err != nil {
		// Reset cache
		c.configCache = nil
		glog.Warning("unable to deploy App Gateway config.", err)
		return errors.New("unable to deploy App Gateway config.")
	}

	glog.V(3).Info("cache: Updated with latest applied config.")
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
