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
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

type ipResource string
type ipAddress string

// MutateAKS applies changes to Kubernetes resources.
func (c AppGwIngressController) MutateAKS() error {
	appGw, cbCtx, err := c.getAppGw()
	if err != nil {
		return err
	}

	// update all relevant ingresses with IP address obtained from existing App Gateway configuration
	cbCtx.IngressList = c.PruneIngress(appGw, cbCtx)
	for _, ingress := range cbCtx.IngressList {
		c.updateIngressStatus(appGw, cbCtx, ingress)
	}
	return nil
}

func (c AppGwIngressController) updateIngressStatus(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingress *v1beta1.Ingress) {
	ips := getIPs(appGw, c.azClient)

	// determine what ipAddress to attach
	usePrivateIP, _ := annotations.UsePrivateIP(ingress)
	usePrivateIP = usePrivateIP || cbCtx.EnvVariables.UsePrivateIP == "true"

	ipConf := appgw.LookupIPConfigurationByType(appGw.FrontendIPConfigurations, usePrivateIP)
	if ipConf == nil {
		return
	}

	if newIP, found := ips[ipResource(*ipConf.ID)]; !found {
		for _, lbi := range ingress.Status.LoadBalancer.Ingress {
			existingIP := lbi.IP
			if existingIP == string(newIP) {
				glog.V(5).Infof("[mutate_aks] IP %s already set on Ingress %s/%s", lbi.IP, ingress.Namespace, ingress.Name)
				return
			}
		}

		if err := c.k8sContext.UpdateIngressStatus(*ingress, k8scontext.IPAddress(newIP)); err != nil {
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonUnableToUpdateIngressStatus, err.Error())
			glog.Errorf("[mutate_aks] Error updating ingress %s/%s IP to %+v", ingress.Namespace, ingress.Name, newIP)
			return
		}
		glog.V(5).Infof("[mutate_aks] Updated Ingress %s/%s IP to %+v", ingress.Namespace, ingress.Name, newIP)
	}
}

func getIPs(appGw *n.ApplicationGateway, azClient azure.AzClient) map[ipResource]ipAddress {
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
	glog.V(5).Infof("[mutate_aks] Found IPs: %+v", ips)
	return ips
}

// getPublicIPAddress gets the ipAddress address associated to public ipAddress on Azure
func getPublicIPAddress(publicIPID string, azClient azure.AzClient) *ipAddress {
	// get public ipAddress
	publicIP, err := azClient.GetPublicIP(publicIPID)
	if err != nil {
		glog.Errorf("[mutate_aks] Unable to get Public IP Address %s. Error %s", publicIPID, err)
		return nil
	}

	ipAddress := ipAddress(*publicIP.IPAddress)
	return &ipAddress
}
