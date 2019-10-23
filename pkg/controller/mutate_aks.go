// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

// MutateAKS applies changes to Kubernetes resources.
func (c AppGwIngressController) MutateAKS(event events.Event) error {
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
