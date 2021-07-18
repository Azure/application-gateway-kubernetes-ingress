// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"sort"

	v1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayloaddistributionpolicy/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

func (c *appGwConfigBuilder) LoadDistributionPolicy(cbCtx *ConfigBuilderContext) error {
	ldps := c.getLoadDistributionPolicies(cbCtx)
	if ldps != nil {
		sort.Sort(sorter.ByLoadDistributionPolicyName(ldps))
	}
	c.appGw.LoadDistributionPolicies = &ldps
	return nil
}

func (c appGwConfigBuilder) getLoadDistributionPolicies(cbCtx *ConfigBuilderContext) []n.ApplicationGatewayLoadDistributionPolicy {
	//memoization, return if cached
	if c.mem.loadDistributionPolicies != nil {
		return *c.mem.loadDistributionPolicies
	}
	appGwLoadDistributionPolicies := []n.ApplicationGatewayLoadDistributionPolicy{}
	//else, traverse all backend pools. if backend is LDP resource, will append it LDP to list
	for backendID := range c.newBackendIdsFiltered(cbCtx) {
		if backendID.isLDPBackend() {
			ldpName := backendID.Backend.Resource.Name
			ldp := c.k8sContext.GetLoadDistributionPolicy(backendID.Namespace, backendID.Backend.Resource.Name)
			ldpResourceName := generateLoadDistributionName(ldpName)
			ldpResourceID := c.appGwIdentifier.LoadDistributionPolicyID(ldpName)
			targets := c.getTargets(backendID, ldp)
			appGWLdp := n.ApplicationGatewayLoadDistributionPolicy{
				Name: &ldpResourceName,
				ID:   &ldpResourceID,
				ApplicationGatewayLoadDistributionPolicyPropertiesFormat: &n.ApplicationGatewayLoadDistributionPolicyPropertiesFormat{
					LoadDistributionAlgorithm: n.RoundRobin,
					LoadDistributionTargets:   &targets,
				},
			}
			appGwLoadDistributionPolicies = append(appGwLoadDistributionPolicies, appGWLdp)
		}
	}
	// set cache and return
	c.mem.loadDistributionPolicies = &appGwLoadDistributionPolicies
	return appGwLoadDistributionPolicies
}

//will create LDP targets for appGW config based on LDP in backendIdentifier
func (c appGwConfigBuilder) getTargets(backendID backendIdentifier, k8sLDP *v1.AzureApplicationGatewayLoadDistributionPolicy) []n.ApplicationGatewayLoadDistributionTarget {
	appGWTargets := make([]n.ApplicationGatewayLoadDistributionTarget, len(k8sLDP.Spec.Targets))
	serviceBackendPortMap := *c.mem.serviceBackendPairsByBackend
	serviceBackendPortPair := serviceBackendPortMap[backendID]
	for _, backend := range k8sLDP.Spec.Targets {
		weight := int32(backend.Weight)
		backendAddressPoolName := generateAddressPoolName(fmt.Sprintf("%v-%v", backendID.Namespace, backend.Service.Name), serviceBackendPortToStr(backend.Service.Port), serviceBackendPortPair.BackendPort)
		backendAddressPoolID := c.appGwIdentifier.AddressPoolID(backendAddressPoolName)
		newTarget := n.ApplicationGatewayLoadDistributionTarget{
			ApplicationGatewayLoadDistributionTargetPropertiesFormat: &n.ApplicationGatewayLoadDistributionTargetPropertiesFormat{
				WeightPerServer: &weight,
				BackendAddressPool: &n.SubResource{
					ID: &backendAddressPoolID,
				},
			},
		}
		appGWTargets = append(appGWTargets, newTarget)
	}
	return appGWTargets
}
