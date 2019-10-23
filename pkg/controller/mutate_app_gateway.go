// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (c AppGwIngressController) getAppGw() (*n.ApplicationGateway, *appgw.ConfigBuilderContext, error) {
	// Get current application gateway config
	appGw, err := c.azClient.GetGateway()
	c.metricStore.IncArmAPICallCounter()
	if err != nil {
		errorLine := fmt.Sprintf("unable to get specified AppGateway [%v], check AppGateway identifier, error=[%v]", c.appGwIdentifier.AppGwName, err)
		glog.Errorf(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonUnableToFetchAppGw, errorLine)
		}
		return nil, nil, ErrFetchingAppGatewayConfig
	}

	cbCtx := &appgw.ConfigBuilderContext{
		ServiceList:  c.k8sContext.ListServices(),
		IngressList:  c.k8sContext.ListHTTPIngresses(),
		EnvVariables: environment.GetEnv(),

		DefaultAddressPoolID:  to.StringPtr(c.appGwIdentifier.AddressPoolID(appgw.DefaultBackendAddressPoolName)),
		DefaultHTTPSettingsID: to.StringPtr(c.appGwIdentifier.HTTPSettingsID(appgw.DefaultBackendHTTPSettingsName)),
	}

	return &appGw, cbCtx, nil
}

// MutateAppGateway applies App Gateway config.
func (c AppGwIngressController) MutateAppGateway(event events.Event) error {
	appGw, cbCtx, err := c.getAppGw()
	if err != nil {
		return err
	}

	existingConfigJSON, _ := dumpSanitizedJSON(appGw, false, to.StringPtr("-- Existing App Gwy Config --"))
	glog.V(5).Info("Existing App Gateway config: ", string(existingConfigJSON))

	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		prohibitedTargets := c.k8sContext.ListAzureProhibitedTargets()
		if len(prohibitedTargets) > 0 {
			cbCtx.ProhibitedTargets = prohibitedTargets
			var prohibitedTargetsList []string
			for _, target := range *brownfield.GetTargetBlacklist(prohibitedTargets) {
				targetJSON, _ := json.Marshal(target)
				prohibitedTargetsList = append(prohibitedTargetsList, string(targetJSON))
			}
			glog.V(3).Infof("[brownfield] Prohibited targets: %s", strings.Join(prohibitedTargetsList, ", "))
		} else {
			glog.Warning("Brownfield Deployment is enabled, but AGIC did not find any AzureProhibitedTarget CRDs; Disabling brownfield deployment feature.")
			cbCtx.EnvVariables.EnableBrownfieldDeployment = false
		}
	}

	if cbCtx.EnvVariables.EnableIstioIntegration {
		istioServices := c.k8sContext.ListIstioVirtualServices()
		istioGateways := c.k8sContext.ListIstioGateways()
		if len(istioGateways) > 0 && len(istioServices) > 0 {
			cbCtx.IstioGateways = istioGateways
			cbCtx.IstioVirtualServices = istioServices
		} else {
			glog.Warning("Istio Integration is enabled, but AGIC needs Istio Gateways and Virtual Services; Disabling Istio integration.")
			cbCtx.EnvVariables.EnableIstioIntegration = false
		}
	}

	cbCtx.IngressList = c.PruneIngress(appGw, cbCtx)

	if cbCtx.EnvVariables.EnableIstioIntegration {
		var gatewaysInfo []string
		for _, gateway := range cbCtx.IstioGateways {
			gatewaysInfo = append(gatewaysInfo, fmt.Sprintf("%s/%s", gateway.Namespace, gateway.Name))
		}
		glog.V(5).Infof("Istio Gateways: %+v", strings.Join(gatewaysInfo, ","))
	}

	// Run fatal validations on the existing config of the Application Gateway.
	if err := appgw.FatalValidateOnExistingConfig(c.recorder, appGw.ApplicationGatewayPropertiesFormat, cbCtx.EnvVariables); err != nil {
		errorLine := fmt.Sprint("Got a fatal validation error on existing Application Gateway config. Will retry getting Application Gateway until error is resolved:", err)
		glog.Error(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonInvalidAppGwConfig, errorLine)
		}
		return err
	}

	// Create a configbuilder based on current appgw config
	configBuilder := appgw.NewConfigBuilder(c.k8sContext, &c.appGwIdentifier, appGw, c.recorder, realClock{})

	// Run validations on the Kubernetes resources which can suggest misconfiguration.
	if err = configBuilder.PreBuildValidate(cbCtx); err != nil {
		errorLine := fmt.Sprint("ConfigBuilder PostBuildValidate returned error:", err)
		glog.Error(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonValidatonError, errorLine)
		}
	}

	var generatedAppGw *n.ApplicationGateway
	// Replace the current appgw config with the generated one
	if generatedAppGw, err = configBuilder.Build(cbCtx); err != nil {
		errorLine := fmt.Sprint("ConfigBuilder Build returned error:", err)
		glog.Error(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonValidatonError, errorLine)
		}
		return err
	}

	// Run post validations to report errors in the config generation.
	if err = configBuilder.PostBuildValidate(cbCtx); err != nil {
		errorLine := fmt.Sprint("ConfigBuilder PostBuildValidate returned error:", err)
		glog.Error(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonValidatonError, errorLine)
		}
	}

	if c.configIsSame(appGw) {
		glog.V(3).Info("cache: Config has NOT changed! No need to connect to ARM.")
		return nil
	}

	glog.V(3).Info("BEGIN AppGateway deployment")
	defer glog.V(3).Info("END AppGateway deployment")

	deploymentStart := time.Now()
	// Initiate deployment
	err = c.azClient.UpdateGateway(generatedAppGw)
	if err != nil {
		// Reset cache
		c.configCache = nil
		configJSON, _ := dumpSanitizedJSON(appGw, cbCtx.EnvVariables.EnableSaveConfigToFile, nil)
		glogIt := glog.Errorf
		if cbCtx.EnvVariables.EnablePanicOnPutError {
			glogIt = glog.Fatalf
		}
		errorLine := fmt.Sprintf("Failed applying App Gwy configuration: %s -- %s", err, string(configJSON))
		glogIt(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonFailedApplyingAppGwConfig, errorLine)
		}
		c.metricStore.IncArmAPIUpdateCallFailureCounter()
		return err
	}
	// Wait until deployment finshes and save the error message
	configJSON, _ := dumpSanitizedJSON(appGw, cbCtx.EnvVariables.EnableSaveConfigToFile, nil)
	glog.V(5).Info(string(configJSON))

	// We keep this at log level 1 to show some heartbeat in the logs. Without this it is way too quiet.
	duration := time.Now().Sub(deploymentStart)
	glog.V(1).Infof("Applied App Gateway config in %+v", duration.String())

	c.metricStore.SetUpdateLatencySec(duration)

	if err != nil {
		// Reset cache
		c.configCache = nil
		errorLine := fmt.Sprint("Unable to deploy App Gateway config.", err)
		glog.Warning(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonFailedApplyingAppGwConfig, errorLine)
		}
		c.metricStore.IncArmAPIUpdateCallFailureCounter()
		return ErrDeployingAppGatewayConfig
	}

	glog.V(3).Info("cache: Updated with latest applied config.")
	c.updateCache(appGw)

	c.metricStore.IncArmAPIUpdateCallSuccessCounter()

	return nil
}
