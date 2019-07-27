// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

// Process is the callback function that will be executed for every event
// in the EventQueue.
func (c AppGwIngressController) Process(event events.Event) error {
	ctx := context.Background()

	// Get current application gateway config
	appGw, err := c.appGwClient.Get(ctx, c.appGwIdentifier.ResourceGroup, c.appGwIdentifier.AppGwName)
	if err != nil {
		glog.Errorf("unable to get specified AppGateway [%v], check AppGateway identifier, error=[%v]", c.appGwIdentifier.AppGwName, err.Error())
		return ErrFetchingAppGatewayConfig
	}

	existingConfigJSON, _ := dumpSanitizedJSON(&appGw, false, to.StringPtr("-- Existing App Gwy Config --"))
	glog.V(5).Info("Existing App Gateway config: ", string(existingConfigJSON))

	envVars := environment.GetEnv()

	cbCtx := &appgw.ConfigBuilderContext{
		ServiceList:           c.k8sContext.ListServices(),
		IngressList:           c.k8sContext.ListHTTPIngresses(),
		EnvVariables:          envVars,
		EnablePanicOnPutError: envVars.EnablePanicOnPutError == "true",
	}

	if envVars.EnableBrownfieldDeployment == "true" {
		prohibitedTargets := c.k8sContext.ListAzureProhibitedTargets()
		if len(prohibitedTargets) > 0 {
			cbCtx.ProhibitedTargets = prohibitedTargets
			cbCtx.EnableBrownfieldDeployment = true
			var prohibitedTargetsList []string
			for _, target := range *brownfield.GetTargetBlacklist(prohibitedTargets) {
				targetJSON, _ := json.Marshal(target)
				prohibitedTargetsList = append(prohibitedTargetsList, string(targetJSON))
			}
			glog.V(3).Infof("[brownfield] Prohibited targets: %s", strings.Join(prohibitedTargetsList, ", "))
		}
	}

	if cbCtx.EnvVariables.EnableIstioIntegration == "true" {
		istioServices := c.k8sContext.ListIstioVirtualServices()
		istioGateways := c.k8sContext.ListIstioGateways()
		if len(istioGateways) > 0 && len(istioServices) > 0 {
			cbCtx.IstioGateways = istioGateways
			cbCtx.IstioVirtualServices = istioServices
			cbCtx.EnableIstioIntegration = true
		}
	}

	cbCtx.IngressList = c.PruneIngress(&appGw, cbCtx)
	if len(cbCtx.IngressList) == 0 {
		errorLine := "no Ingress in the pruned Ingress list. Please check Ingress events to get more information"
		glog.Error(errorLine)
		return nil
	}

	if cbCtx.EnableIstioIntegration {
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

	// Create a configbuilder based on current appgw config
	configBuilder := appgw.NewConfigBuilder(c.k8sContext, &c.appGwIdentifier, &appGw, c.recorder)

	// Run validations on the Kubernetes resources which can suggest misconfiguration.
	if err = configBuilder.PreBuildValidate(cbCtx); err != nil {
		glog.Error("ConfigBuilder PostBuildValidate returned error:", err)
	}

	var generatedAppGw *n.ApplicationGateway
	// Replace the current appgw config with the generated one
	if generatedAppGw, err = configBuilder.Build(cbCtx); err != nil {
		glog.Error("ConfigBuilder Build returned error:", err)
		return err
	}

	// Run post validations to report errors in the config generation.
	if err = configBuilder.PostBuildValidate(cbCtx); err != nil {
		glog.Error("ConfigBuilder PostBuildValidate returned error:", err)
	}

	if c.configIsSame(&appGw) {
		glog.V(3).Info("cache: Config has NOT changed! No need to connect to ARM.")
		return nil
	}

	glog.V(3).Info("BEGIN AppGateway deployment")
	defer glog.V(3).Info("END AppGateway deployment")

	logToFile := cbCtx.EnvVariables.EnableSaveConfigToFile == "true"

	deploymentStart := time.Now()
	// Initiate deployment
	appGwFuture, err := c.appGwClient.CreateOrUpdate(ctx, c.appGwIdentifier.ResourceGroup, c.appGwIdentifier.AppGwName, *generatedAppGw)
	if err != nil {
		// Reset cache
		c.configCache = nil
		configJSON, _ := dumpSanitizedJSON(&appGw, logToFile, nil)
		glogIt := glog.Errorf
		if cbCtx.EnablePanicOnPutError {
			glogIt = glog.Fatalf
		}
		glogIt("Failed applying App Gwy configuration: %s -- %s", err, string(configJSON))
		return err
	}
	// Wait until deployment finshes and save the error message
	err = appGwFuture.WaitForCompletionRef(ctx, c.appGwClient.BaseClient.Client)
	configJSON, _ := dumpSanitizedJSON(&appGw, logToFile, nil)
	glog.V(5).Info(string(configJSON))

	// We keep this at log level 1 to show some heartbeat in the logs. Without this it is way too quiet.
	glog.V(1).Infof("Applied App Gateway config in %+v", time.Now().Sub(deploymentStart).String())

	if err != nil {
		// Reset cache
		c.configCache = nil
		glog.Warning("Unable to deploy App Gateway config.", err)
		return ErrDeployingAppGatewayConfig
	}

	glog.V(3).Info("cache: Updated with latest applied config.")
	c.updateCache(&appGw)

	return nil
}
