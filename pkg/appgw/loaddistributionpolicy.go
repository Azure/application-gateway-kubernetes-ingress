// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"sort"

	v1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayloaddistributionpolicy/v1beta1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
	"github.com/Azure/go-autorest/autorest/to"
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
	appGwLoadDistributionPoliciesMap := make(map[string]n.ApplicationGatewayLoadDistributionPolicy)
	//else, traverse all backend pools. if backend is LDP resource, will append it LDP to list
	for backendID := range c.newBackendIdsFiltered(cbCtx) {
		if backendID.isLDPBackend() {
			ldpName := backendID.Backend.Resource.Name
			ldp, err := c.k8sContext.GetLoadDistributionPolicy(backendID.Namespace, backendID.Backend.Resource.Name)
			if err != nil {
				continue
			}
			//NEED TO ADD NS
			ldpResourceName := generateLoadDistributionName(backendID.Namespace, ldpName)
			ldpResourceID := c.appGwIdentifier.LoadDistributionPolicyID(ldpResourceName)
			targets := c.getTargets(backendID, ldp)
			appGWLdp := n.ApplicationGatewayLoadDistributionPolicy{
				Name: &ldpResourceName,
				ID:   &ldpResourceID,
				ApplicationGatewayLoadDistributionPolicyPropertiesFormat: &n.ApplicationGatewayLoadDistributionPolicyPropertiesFormat{
					LoadDistributionAlgorithm: n.RoundRobin,
					LoadDistributionTargets:   &targets,
				},
			}
			appGwLoadDistributionPoliciesMap[ldpName] = appGWLdp
		}
	}

	appGwLoadDistributionPolicies := []n.ApplicationGatewayLoadDistributionPolicy{}

	for _, ldp := range appGwLoadDistributionPoliciesMap {
		appGwLoadDistributionPolicies = append(appGwLoadDistributionPolicies, ldp)
	}

	// set cache and return
	c.mem.loadDistributionPolicies = &appGwLoadDistributionPolicies
	return appGwLoadDistributionPolicies
}

//will create LDP targets for appGW config based on LDP in backendIdentifier
func (c appGwConfigBuilder) getTargets(backendID backendIdentifier, k8sLDP *v1.AzureApplicationGatewayLoadDistributionPolicy) []n.ApplicationGatewayLoadDistributionTarget {
	var appGWTargets []n.ApplicationGatewayLoadDistributionTarget
	serviceBackendPortMap := *c.mem.serviceBackendPairsByBackend
	serviceBackendPortPair := serviceBackendPortMap[backendID]
	for targetIdx, target := range k8sLDP.Spec.Targets {
		weight := int32(target.Weight)
		backendAddressPoolName := generateAddressPoolName(fmt.Sprintf("%v-%v", backendID.Namespace, target.Backend.Service.Name), serviceBackendPortToStr(target.Backend.Service.Port), serviceBackendPortPair.BackendPort)
		backendAddressPoolID := c.appGwIdentifier.AddressPoolID(backendAddressPoolName)
		targetName := fmt.Sprint(k8sLDP.Name, "-target-", targetIdx)
		newTarget := n.ApplicationGatewayLoadDistributionTarget{
			Etag: to.StringPtr("*"),
			Name: to.StringPtr(targetName),
			ID:   to.StringPtr(c.appGwIdentifier.ldpTargetID(k8sLDP.Name, targetName)),
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
