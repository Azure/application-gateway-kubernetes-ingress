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

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

// Process is the callback function that will be executed for every event
// in the EventQueue.
func (c AppGwIngressController) Process(event events.Event) error {
	// Get current application gateway config
	appGw, err := c.azClient.GetGateway()
	c.metricStore.IncArmAPICallCounter()
	if err != nil {
		glog.Errorf("unable to get specified AppGateway [%v], check AppGateway identifier, error=[%v]", c.appGwIdentifier.AppGwName, err)
		return ErrFetchingAppGatewayConfig
	}

	c.updateIPAddressMap(&appGw)

	existingConfigJSON, _ := dumpSanitizedJSON(&appGw, false, to.StringPtr("-- Existing App Gwy Config --"))
	glog.V(5).Info("Existing App Gateway config: ", string(existingConfigJSON))

	cbCtx := &appgw.ConfigBuilderContext{
		ServiceList:  c.k8sContext.ListServices(),
		IngressList:  c.k8sContext.ListHTTPIngresses(),
		EnvVariables: environment.GetEnv(),

		DefaultAddressPoolID:  to.StringPtr(c.appGwIdentifier.AddressPoolID(appgw.DefaultBackendAddressPoolName)),
		DefaultHTTPSettingsID: to.StringPtr(c.appGwIdentifier.HTTPSettingsID(appgw.DefaultBackendHTTPSettingsName)),
	}

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

	cbCtx.IngressList = c.PruneIngress(&appGw, cbCtx)
	if len(cbCtx.IngressList) == 0 && !cbCtx.EnvVariables.EnableIstioIntegration {
		errorLine := "no Ingress in the pruned Ingress list. Please check Ingress events to get more information"
		glog.Error(errorLine)
		return nil
	}

	if cbCtx.EnvVariables.EnableIstioIntegration {
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
		// update ingresses with appgw gateway ip address
		c.updateIngressStatus(generatedAppGw, cbCtx, event)

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
		configJSON, _ := dumpSanitizedJSON(&appGw, cbCtx.EnvVariables.EnableSaveConfigToFile, nil)
		glogIt := glog.Errorf
		if cbCtx.EnvVariables.EnablePanicOnPutError {
			glogIt = glog.Fatalf
		}
		glogIt("Failed applying App Gwy configuration: %s -- %s", err, string(configJSON))
		c.metricStore.IncArmAPIUpdateCallFailureCounter()
		return err
	}
	// Wait until deployment finshes and save the error message
	configJSON, _ := dumpSanitizedJSON(&appGw, cbCtx.EnvVariables.EnableSaveConfigToFile, nil)
	glog.V(5).Info(string(configJSON))

	// We keep this at log level 1 to show some heartbeat in the logs. Without this it is way too quiet.
	duration := time.Now().Sub(deploymentStart)
	glog.V(1).Infof("Applied App Gateway config in %+v", duration.String())

	c.metricStore.SetUpdateLatencySec(duration)

	if err != nil {
		// Reset cache
		c.configCache = nil
		glog.Warning("Unable to deploy App Gateway config.", err)
		c.metricStore.IncArmAPIUpdateCallFailureCounter()
		return ErrDeployingAppGatewayConfig
	}

	glog.V(3).Info("cache: Updated with latest applied config.")
	c.updateCache(&appGw)

	// update ingresses with appgw gateway ip address
	c.updateIngressStatus(generatedAppGw, cbCtx, event)

	c.metricStore.IncArmAPIUpdateCallSuccessCounter()

	return nil
}

func (c AppGwIngressController) updateIngressStatus(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, event events.Event) {
	ingress, ok := event.Value.(*v1beta1.Ingress)
	if !ok {
		return
	}

	// check if this ingress is for AGIC or not, it might have been updated
	if !k8scontext.IsIngressApplicationGateway(ingress) || !cbCtx.InIngressList(ingress) {
		if err := c.k8sContext.UpdateIngressStatus(*ingress, ""); err != nil {
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonUnableToUpdateIngressStatus, err.Error())
		}
		return
	}

	// determine what ip to attach
	usePrivateIP, _ := annotations.UsePrivateIP(ingress)
	usePrivateIP = usePrivateIP || cbCtx.EnvVariables.UsePrivateIP == "true"
	if ipConf := appgw.LookupIPConfigurationByType(appGw.FrontendIPConfigurations, usePrivateIP); ipConf != nil {
		if ipAddress, ok := c.ipAddressMap[*ipConf.ID]; ok {
			if err := c.k8sContext.UpdateIngressStatus(*ingress, ipAddress); err != nil {
				c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonUnableToUpdateIngressStatus, err.Error())
			}
		}
	}
}

func (c AppGwIngressController) updateIPAddressMap(appGw *n.ApplicationGateway) {
	for _, ipConf := range *appGw.FrontendIPConfigurations {
		if _, ok := c.ipAddressMap[*ipConf.ID]; ok {
			return
		}

		if ipConf.PrivateIPAddress != nil {
			c.ipAddressMap[*ipConf.ID] = k8scontext.IPAddress(*ipConf.PrivateIPAddress)
		} else if ipAddress := c.getPublicIPAddress(*ipConf.PublicIPAddress.ID); ipAddress != nil {
			c.ipAddressMap[*ipConf.ID] = *ipAddress
		}
	}
}

// getPublicIPAddress gets the ip address associated to public ip on Azure
func (c AppGwIngressController) getPublicIPAddress(publicIPID string) *k8scontext.IPAddress {
	// get public ip
	publicIP, err := c.azClient.GetPublicIP(publicIPID)
	if err != nil {
		glog.Errorf("Unable to get Public IP Address %s. Error %s", publicIPID, err)
		return nil
	}

	ipAddress := k8scontext.IPAddress(*publicIP.IPAddress)
	return &ipAddress
}
