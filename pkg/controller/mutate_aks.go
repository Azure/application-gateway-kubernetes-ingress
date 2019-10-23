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

type ipResource string
type ip string

// MutateAKS applies changes to Kubernetes resources.
func (c AppGwIngressController) MutateAKS(event events.Event) error {
	appGw, cbCtx, err := c.getAppGw()
	if err != nil {
		return err
	}

	if ingress, ok := event.Value.(*v1beta1.Ingress); ok {
		// update ingresses with appgw gateway ip address
		c.updateIngressStatus(appGw, cbCtx, ingress)

	}
	return nil
}

func (c AppGwIngressController) updateIngressStatus(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingress *v1beta1.Ingress) {
	// check if this ingress is for AGIC or not, it might have been updated
	if !k8scontext.IsIngressApplicationGateway(ingress) || !cbCtx.InIngressList(ingress) {
		if err := c.k8sContext.UpdateIngressStatus(*ingress, ""); err != nil {
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonUnableToUpdateIngressStatus, err.Error())
		}
		return
	}

	ips := c.getIPs(appGw)

	// determine what ip to attach
	usePrivateIP, _ := annotations.UsePrivateIP(ingress)
	usePrivateIP = usePrivateIP || cbCtx.EnvVariables.UsePrivateIP == "true"
	if ipConf := appgw.LookupIPConfigurationByType(appGw.FrontendIPConfigurations, usePrivateIP); ipConf != nil {
		if ipAddress, ok := ips[ipResource(*ipConf.ID)]; ok {
			for _, lbi := range ingress.Status.LoadBalancer.Ingress {
				if lbi.IP == string(ipAddress) {
					glog.V(5).Infof("IP %s already set on Ingress %s/%s", lbi.IP, ingress.Namespace, ingress.Name)
					return
				}
			}

			if err := c.k8sContext.UpdateIngressStatus(*ingress, k8scontext.IPAddress(ipAddress)); err != nil {
				c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonUnableToUpdateIngressStatus, err.Error())
			}
		}
	}
}

func (c AppGwIngressController) getIPs(appGw *n.ApplicationGateway) map[ipResource]ip {
	ips := make(map[ipResource]ip)
	for _, ipConf := range *appGw.FrontendIPConfigurations {
		ipID := ipResource(*ipConf.ID)
		if _, ok := ips[ipID]; ok {
			continue
		}

		if ipConf.PrivateIPAddress != nil {
			ips[ipID] = ip(*ipConf.PrivateIPAddress)
		} else if ipAddress := c.getPublicIPAddress(*ipConf.PublicIPAddress.ID); ipAddress != nil {
			ips[ipID] = *ipAddress
		}
	}
	return ips
}

// getPublicIPAddress gets the ip address associated to public ip on Azure
func (c AppGwIngressController) getPublicIPAddress(publicIPID string) *ip {
	// get public ip
	publicIP, err := c.azClient.GetPublicIP(publicIPID)
	if err != nil {
		glog.Errorf("Unable to get Public IP Address %s. Error %s", publicIPID, err)
		return nil
	}

	ipAddress := ip(*publicIP.IPAddress)
	return &ipAddress
}
