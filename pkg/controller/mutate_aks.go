// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"fmt"

	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

type ipResource string
type ipAddress string

// MutateAllIngress applies changes to ingress status object in kubernetes
func (c AppGwIngressController) MutateAllIngress(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext) error {
	ips := getIPsFromAppGateway(appGw, c.azClient)

	// update all relevant ingresses with IP address obtained from existing App Gateway configuration
	cbCtx.IngressList = c.PruneIngress(appGw, cbCtx)
	for _, ingress := range cbCtx.IngressList {
		c.updateIngressStatus(appGw, cbCtx, ingress, ips)
	}
	return nil
}

// ResetAllIngress resets the ingress status object in kubernetes
func (c AppGwIngressController) ResetAllIngress(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext) {
	for _, ingress := range cbCtx.IngressList {
		if err := c.k8sContext.UpdateIngressStatus(*ingress, k8scontext.IPAddress("")); err != nil {
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonUnableToResetIngressStatus, err.Error())
			klog.Errorf("[mutate_aks] Error resetting ingress %s/%s IP", ingress.Namespace, ingress.Name)
			continue
		}

		msg := fmt.Sprintf("Reset IP for Ingress %s/%s. Application Gateway %s is in stopped state", ingress.Namespace, ingress.Name, *appGw.ID)
		c.recorder.Event(ingress, v1.EventTypeNormal, events.ReasonResetIngressStatus, msg)
		klog.V(5).Infof(msg)
	}
}

func (c AppGwIngressController) updateIngressStatus(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingress *networking.Ingress, ips map[ipResource]ipAddress) {

	// determine what ipAddress to attach
	usePrivateIP, _ := annotations.UsePrivateIP(ingress)
	usePrivateIP = usePrivateIP || cbCtx.EnvVariables.UsePrivateIP

	ipConf := appgw.LookupIPConfigurationByType(appGw.FrontendIPConfigurations, usePrivateIP)
	if ipConf == nil {
		klog.V(9).Info("[mutate_aks] No IP config for App Gwy: ", appGw.Name)
		return
	}

	klog.V(5).Infof("[mutate_aks] Resolving IP for ID (%s)", *ipConf.ID)
	if newIP, found := ips[ipResource(*ipConf.ID)]; found {
		if err := c.k8sContext.UpdateIngressStatus(*ingress, k8scontext.IPAddress(newIP)); err != nil {
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonUnableToUpdateIngressStatus, err.Error())
			klog.Errorf("[mutate_aks] Error updating ingress %s/%s IP to %+v", ingress.Namespace, ingress.Name, newIP)
			return
		}
		klog.V(5).Infof("[mutate_aks] Updated Ingress %s/%s IP to %+v", ingress.Namespace, ingress.Name, newIP)
	}
}

func getIPsFromAppGateway(appGw *n.ApplicationGateway, azClient azure.AzClient) map[ipResource]ipAddress {
	ips := make(map[ipResource]ipAddress)
	for _, ipConf := range *appGw.FrontendIPConfigurations {
		ipID := ipResource(*ipConf.ID)
		if _, ok := ips[ipID]; ok {
			continue
		}

		if ipConf.PrivateIPAddress != nil {
			ips[ipID] = ipAddress(*ipConf.PrivateIPAddress)
		} else if ipAddress := getPublicIPAddress(*ipConf.PublicIPAddress.ID, azClient); ipAddress != nil {
			ips[ipID] = *ipAddress
		}
	}
	klog.V(5).Infof("[mutate_aks] Found IPs: %+v", ips)
	return ips
}

// getPublicIPAddress gets the ipAddress address associated to public ipAddress on Azure
func getPublicIPAddress(publicIPID string, azClient azure.AzClient) *ipAddress {
	// get public ipAddress
	publicIP, err := azClient.GetPublicIP(publicIPID)
	if err != nil {
		klog.Errorf("[mutate_aks] Unable to get Public IP Address %s. Error %s", publicIPID, err)
		return nil
	}

	ipAddress := ipAddress(*publicIP.IPAddress)
	return &ipAddress
}
