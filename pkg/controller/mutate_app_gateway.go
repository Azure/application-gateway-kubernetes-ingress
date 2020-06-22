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

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"

	agpoolv1beta1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewaybackendpool/v1beta1"

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
		glog.Errorf(e.Error())
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
	glog.V(5).Info("Existing App Gateway config: ", string(existingConfigJSON))
	existingBackendAddressPools := *appGw.ApplicationGatewayPropertiesFormat.BackendAddressPools
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
	// -------------------------- //

	// Generate App Gateway Phase //
	// -------------------------- //
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
	// -------------------------- //

	// Post Compare Phase //
	// ------------------ //

	// (TO-DO): a global flag to enable or disable the fast path, the flag shall be enabled by default once we start to migrate to fast update
	if cbCtx.EnvVariables.CCPEnabled {
		generatedBackendAddressPools := generatedAppGw.ApplicationGatewayPropertiesFormat.BackendAddressPools
		if c.isBackendAddressPoolsUpdated(generatedBackendAddressPools, &existingBackendAddressPools) {
			glog.V(3).Info("Backend address pool is updated")
			// Endpoint event has been verified at this point
			if _, yes := event.Value.(*v1.Endpoints); !yes {
				// if the event is not Endpoint event, backendPool change will not be applied
				glog.V(3).Info("Not endpoint event, skip to apply backend address pool changes")
				// update BackendAddressPools but not assign backendAddresses
				for _, backendAddressPool := range *generatedBackendAddressPools {
					backendAddressPool.ApplicationGatewayBackendAddressPoolPropertiesFormat.BackendAddresses = nil
				}

			} else {
				// otherwise, we start to update our CRD
				glog.V(3).Info("Endpoint event identified, start to update backend address pool")
				// check crd by name
				AddressPoolCRDObjectID := c.appGwIdentifier.BackendAddressPoolCRDObjectID()
				backendPool, err := c.k8sContext.GetBackendPool(AddressPoolCRDObjectID)
				if err != nil {
					glog.Warningf("Cannot find address pool CRD object Id: %s, a CRD object will be initialized, backend address pool update falls back to slow-update!", AddressPoolCRDObjectID)
					// TO-DO: Initialize a CRD object if it doesn't exist
					initBackendPool := agpoolv1beta1.AzureApplicationGatewayBackendPool{}
					initBackendPool.Name = AddressPoolCRDObjectID
					if _, err := c.k8sContext.CreateBackendPool(&initBackendPool); err != nil {
						e := controllererrors.NewError(
							controllererrors.ErrorInitializeBackendAddressPool,
							"Unable to initialize backend address pool",
						)
						glog.Warning(e.Error())
					}

					// fallback to slow update to apply in case of failure
					// generate metric for slow-update count
					c.MetricStore.IncAddressPoolSlowUpdateCounter()
				} else {
					glog.V(3).Infof("Find AzureApplicationGatewayBackendPool CRD object id: %s", AddressPoolCRDObjectID)
					if generatedBackendAddressPools == nil {
						e := controllererrors.NewError(
							controllererrors.ErrorNoBackendAddressPool,
							"Unable to find any address pool from backend",
						)
						glog.Error(e.Error())
						return e
					}

					// reset crd before update
					backendPool.Spec.BackendAddressPools = []agpoolv1beta1.BackendAddressPool{}
					var updatedBackendAddressPools []agpoolv1beta1.BackendAddressPool
					// apply updates to CRD
					for _, backendAddressPool := range *generatedBackendAddressPools {
						pool := agpoolv1beta1.BackendAddressPool{
							Name:             *backendAddressPool.ID,
							BackendAddresses: c.getIPAddresses(backendAddressPool.BackendAddresses),
						}
						updatedBackendAddressPools = append(updatedBackendAddressPools, pool)
					}

					backendPool.Spec.BackendAddressPools = updatedBackendAddressPools
					if _, err := c.k8sContext.UpdateBackendPool(backendPool); err != nil {
						glog.Warningf("Failed to update address pool CRD object Id: %s, backend address pool update falls back to slow-update!", AddressPoolCRDObjectID)
						c.MetricStore.IncAddressPoolSlowUpdateCounter()
					} else {
						for _, obj := range backendPool.Spec.BackendAddressPools {
							var ips []string
							for _, address := range obj.BackendAddresses {
								ips = append(ips, address.IPAddress)
							}
							glog.V(9).Infof("Backend pool ID: %s, IPs: %s", obj.Name, strings.Join(ips, ","))
						}

						// make sure no backendaddress will be passed through slow path
						// TO-DO: use the code once we have feature flag for ccp
						// for _, backendAddressPool := range *generatedBackendAddressPools {
						// 	backendAddressPool.ApplicationGatewayBackendAddressPoolPropertiesFormat.BackendAddresses = nil
						// }
					}

				}
			}
		} else {
			glog.V(3).Info("Backend address pool has NOT been changed!")
		}
	}

	// if this is not a reconciliation task
	// then compare the generated state with cached state
	if event.Type != events.PeriodicReconcile {
		if c.configIsSame(appGw) {
			glog.V(3).Info("cache: Config has NOT changed! No need to connect to ARM.")
			return nil
		}
	}
	// ------------------ //

	// Deployment Phase //
	// ---------------- //

	configJSON, _ := dumpSanitizedJSON(appGw, cbCtx.EnvVariables.EnableSaveConfigToFile, nil)
	glog.V(5).Infof("Generated config:\n%s", string(configJSON))

	// Initiate deployment
	glog.V(3).Info("BEGIN AppGateway deployment")
	defer glog.V(3).Info("END AppGateway deployment")
	err = c.azClient.UpdateGateway(generatedAppGw)
	if err != nil {
		// Reset cache
		c.configCache = nil
		return err
	}
	glog.V(1).Infof("Applied generated Application Gateway configuration")
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

	glog.V(3).Info("cache: Updated with latest applied config.")
	c.updateCache(appGw)
	// ----------- //

	return nil
}
