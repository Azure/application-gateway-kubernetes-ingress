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

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// GetAppGw gets App Gateway config.
func (c AppGwIngressController) GetAppGw() (*n.ApplicationGateway, *appgw.ConfigBuilderContext, error) {
	// Get current application gateway config
	appGw, err := c.azClient.GetGateway()
	c.MetricStore.IncArmAPICallCounter()
	if err != nil {
		e := controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorFetchingAppGatewayConfig,
			err,
			"unable to get specified AppGateway [%v], check AppGateway identifier", c.appGwIdentifier.AppGwName,
		)
		klog.Errorf(e.Error())
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonUnableToFetchAppGw, e.Error())
		}
		return nil, nil, e
	}

	cbCtx := &appgw.ConfigBuilderContext{
		ServiceList:  c.k8sContext.ListServices(),
		IngressList:  c.k8sContext.ListHTTPIngresses(),
		EnvVariables: environment.GetEnv(),

		DefaultAddressPoolID:  to.StringPtr(c.appGwIdentifier.AddressPoolID(appgw.DefaultBackendAddressPoolName)),
		DefaultHTTPSettingsID: to.StringPtr(c.appGwIdentifier.HTTPSettingsID(appgw.DefaultBackendHTTPSettingsName)),

		ExistingPortsByNumber: make(map[appgw.Port]n.ApplicationGatewayFrontendPort),
	}

	for _, port := range *appGw.FrontendPorts {
		cbCtx.ExistingPortsByNumber[appgw.Port(*port.Port)] = port
	}

	return &appGw, cbCtx, nil
}

// MutateAppGateway applies App Gateway config.
func (c AppGwIngressController) MutateAppGateway(event events.Event, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext) error {
	var err error
	existingConfigJSON, _ := dumpSanitizedJSON(appGw, false, to.StringPtr("-- Existing App Gwy Config --"))
	klog.V(5).Info("Existing App Gateway config: ", string(existingConfigJSON))

	// Prepare k8s resources Phase //
	// --------------------------- //
	if cbCtx.EnvVariables.EnableBrownfieldDeployment {
		prohibitedTargets := c.k8sContext.ListAzureProhibitedTargets()
		if len(prohibitedTargets) > 0 {
			cbCtx.ProhibitedTargets = prohibitedTargets
			var prohibitedTargetsList []string
			for _, target := range *brownfield.GetTargetBlacklist(prohibitedTargets) {
				targetJSON, _ := json.Marshal(target)
				prohibitedTargetsList = append(prohibitedTargetsList, string(targetJSON))
			}
			klog.V(3).Infof("[brownfield] Prohibited targets: %s", strings.Join(prohibitedTargetsList, ", "))
		} else {
			klog.Warning("Brownfield Deployment is enabled, but AGIC did not find any AzureProhibitedTarget CRDs; Disabling brownfield deployment feature.")
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
			klog.Warning("Istio Integration is enabled, but AGIC needs Istio Gateways and Virtual Services; Disabling Istio integration.")
			cbCtx.EnvVariables.EnableIstioIntegration = false
		}
	}

	cbCtx.IngressList = c.PruneIngress(appGw, cbCtx)

	if cbCtx.EnvVariables.EnableIstioIntegration {
		var gatewaysInfo []string
		for _, gateway := range cbCtx.IstioGateways {
			gatewaysInfo = append(gatewaysInfo, fmt.Sprintf("%s/%s", gateway.Namespace, gateway.Name))
		}
		klog.V(5).Infof("Istio Gateways: %+v", strings.Join(gatewaysInfo, ","))
	}

	// Generate App Gateway Phase //
	// -------------------------- //
	// Create a configbuilder based on current appgw config
	configBuilder := appgw.NewConfigBuilder(c.k8sContext, &c.appGwIdentifier, appGw, c.recorder, realClock{})

	// Run validations on the Kubernetes resources which can suggest misconfiguration.
	if err = configBuilder.PreBuildValidate(cbCtx); err != nil {
		errorLine := fmt.Sprint("ConfigBuilder PostBuildValidate returned error:", err)
		klog.Error(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonValidatonError, errorLine)
		}
	}

	var generatedAppGw *n.ApplicationGateway
	// Replace the current appgw config with the generated one
	if generatedAppGw, err = configBuilder.Build(cbCtx); err != nil {
		errorLine := fmt.Sprint("ConfigBuilder Build returned error:", err)
		klog.Error(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonValidatonError, errorLine)
		}
		return err
	}

	// Run post validations to report errors in the config generation.
	if err = configBuilder.PostBuildValidate(cbCtx); err != nil {
		errorLine := fmt.Sprint("ConfigBuilder PostBuildValidate returned error:", err)
		klog.Error(errorLine)
		if c.agicPod != nil {
			c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonValidatonError, errorLine)
		}
	}
	// -------------------------- //

	// Post Compare Phase //
	// ------------------ //
	// if this is not a reconciliation task
	// then compare the generated state with cached state
	if event.Type != events.PeriodicReconcile {
		if c.configIsSame(appGw) {
			klog.V(3).Info("cache: Config has NOT changed! No need to connect to ARM.")
			return nil
		}
	}
	// ------------------ //

	// Deployment Phase //
	// ---------------- //

	configJSON, _ := dumpSanitizedJSON(appGw, cbCtx.EnvVariables.EnableSaveConfigToFile, nil)
	klog.V(5).Infof("Generated config:\n%s", string(configJSON))

	// Initiate deployment
	klog.V(3).Info("BEGIN AppGateway deployment")
	defer klog.V(3).Info("END AppGateway deployment")
	err = c.azClient.UpdateGateway(generatedAppGw)
	if err != nil {
		// Reset cache
		c.configCache = nil
		return err
	}
	klog.V(1).Infof("Applied generated Application Gateway configuration")
	// ----------------- //

	// Cache Phase //
	// ----------- //
	if err != nil {
		// Reset cache
		c.configCache = nil
		return controllererrors.NewErrorWithInnerErrorf(
			controllererrors.ErrorDeployingAppGatewayConfig,
			err,
			"unable to get specified AppGateway %s", c.appGwIdentifier.AppGwName,
		)
	}

	klog.V(3).Info("cache: Updated with latest applied config.")
	c.updateCache(appGw)
	// ----------- //

	return nil
}
