// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test blacklisting backend pools", func() {

	listeners := []n.ApplicationGatewayHTTPListener{
		*fixtures.GetDefaultListener(),
		*fixtures.GetListenerPathBased1(),
		*fixtures.GetListenerBasic(),
	}

	routingRules := []n.ApplicationGatewayRequestRoutingRule{
		*fixtures.GetDefaultRoutingRule(),
		*fixtures.GetRequestRoutingRulePathBased1(),
		*fixtures.GetRequestRoutingRuleBasic(),
	}

	paths := []n.ApplicationGatewayURLPathMap{
		*fixtures.GetURLPathMap1(),
		*fixtures.GetDefaultURLPathMap(),
	}

	defaultPool := fixtures.GetDefaultBackendPool()
	pool1 := fixtures.GetBackendPool1()
	pool2 := fixtures.GetBackendPool2()
	pool3 := fixtures.GetBackendPool3()

	// Create a list of pools
	pools := []n.ApplicationGatewayBackendAddressPool{
		defaultPool,
		pool1, // managed
		pool2, // managed
		pool3, // prohibited
	}

	appGw := n.ApplicationGateway{
		ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
			HTTPListeners:       &listeners,
			RequestRoutingRules: &routingRules,
			URLPathMaps:         &paths,
			BackendAddressPools: &pools,
		},
	}

	prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets()

	brownfieldContext := NewExistingResources(appGw, prohibitedTargets, nil, &defaultPool)

	prohibitWildcard := &ptv1.AzureIngressProhibitedTarget{
		Spec: ptv1.AzureIngressProhibitedTargetSpec{},
	}

	Context("Test MergePools()", func() {

		poolList0 := []n.ApplicationGatewayBackendAddressPool{
			pool1,
		}

		poolList1 := []n.ApplicationGatewayBackendAddressPool{
			pool1,
			pool2,
		}

		poolList2 := []n.ApplicationGatewayBackendAddressPool{
			pool2,
		}

		It("should be able to merge lists of pools", func() {
			merge1 := MergePools(poolList1, poolList2)
			Expect(len(merge1)).To(Equal(2))
			Expect(merge1).To(ContainElement(pool1))
			Expect(merge1).To(ContainElement(pool2))

			merge2 := MergePools(poolList0, poolList2)
			Expect(len(merge2)).To(Equal(2))
			Expect(merge1).To(ContainElement(pool1))
			Expect(merge1).To(ContainElement(pool2))
		})
	})

	Context("Test GetBlacklistedPools()", func() {

		It("Should determine what pools are blacklisted", func() {
			blacklisted, notBlacklisted := brownfieldContext.GetBlacklistedPools()

			Expect(len(blacklisted)).To(Equal(2))
			Expect(blacklisted).To(ContainElement(pool1))
			Expect(blacklisted).To(ContainElement(pool2))

			// Default backend pool is NOT in the blacklist.
			Expect(blacklisted).ToNot(ContainElement(defaultPool))

			// This pool is not linked to any listeners so we leave it alone.
			Expect(blacklisted).ToNot(ContainElement(pool3))

			// --- inverse check

			Expect(len(notBlacklisted)).To(Equal(2))
			Expect(notBlacklisted).ToNot(ContainElement(pool1))
			Expect(notBlacklisted).ToNot(ContainElement(pool2))

			// Default backend pool is NOT in the blacklist.
			Expect(notBlacklisted).To(ContainElement(defaultPool))

			// This pool is not linked to any listeners so we leave it alone.
			Expect(notBlacklisted).To(ContainElement(pool3))
		})
	})

	Context("Test GetBlacklistedPools() with everyhting blacklisted", func() {

		It("blacklists everything linked to a listener", func() {
			prohibitedTargets := append(fixtures.GetAzureIngressProhibitedTargets(), prohibitWildcard)
			bfCtx := NewExistingResources(appGw, prohibitedTargets, nil, &defaultPool)
			blacklisted, notBlacklisted := bfCtx.GetBlacklistedPools()

			Expect(len(blacklisted)).To(Equal(3))
			Expect(blacklisted).To(ContainElement(pool1))
			Expect(blacklisted).To(ContainElement(pool2))
			Expect(blacklisted).To(ContainElement(pool2))

			// Default backend pool is blacklisted under the prohibitWildcard target.
			Expect(blacklisted).To(ContainElement(defaultPool))

			// This is a backendpool that is not connected to any listener so we leave it alone
			Expect(blacklisted).ToNot(ContainElement(pool3))

			// --- inverse check

			Expect(len(notBlacklisted)).To(Equal(1))
			Expect(notBlacklisted).ToNot(ContainElement(pool1))
			Expect(notBlacklisted).ToNot(ContainElement(pool2))

			// Default backend pool is blacklisted under the prohibitWildcard target.
			Expect(notBlacklisted).ToNot(ContainElement(defaultPool))

			// This is a backendpool that is not connected to any listener so we leave it alone
			Expect(notBlacklisted).To(ContainElement(pool3))
		})
	})
})
