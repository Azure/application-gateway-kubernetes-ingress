// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test blacklisting backend pools", func() {

	listeners := []n.ApplicationGatewayHTTPListener{
		*fixtures.GetListener1(),
		*fixtures.GetListener2(),
	}

	routingRules := []n.ApplicationGatewayRequestRoutingRule{
		*fixtures.GetRequestRoutingRulePathBased(),
		*fixtures.GetRequestRoutingRuleBasic(),
	}

	paths := []n.ApplicationGatewayURLPathMap{
		*fixtures.GeURLPathMap(),
	}

	pool1 := fixtures.GetBackendPool1()
	pool2 := fixtures.GetBackendPool2()
	pool3 := fixtures.GetBackendPool3()

	// Create a list of pools
	pools := []n.ApplicationGatewayBackendAddressPool{
		pool1, // managed
		pool2, // managed
		pool3, // prohibited
	}

	brownfieldContext := PoolContext{
		Listeners:    listeners,
		RoutingRules: routingRules,
		PathMaps:     paths,
		BackendPools: pools,
	}

	prohibitedTargets := fixtures.GetAzureIngressProhibitedTargets()

	Context("Test getPoolToTargetsMap", func() {

		actual := brownfieldContext.getPoolToTargetsMap()

		It("should have created map of pool name to list of targets", func() {
			expected := poolToTargets{
				fixtures.BackendAddressPoolName1: {
					{
						Hostname: tests.Host,
						Path:     fixtures.PathFoo,
					},

					{
						Hostname: tests.Host,
						Path:     fixtures.PathBar,
					},

					{
						Hostname: tests.Host,
						Path:     fixtures.PathBaz,
					},
				},

				fixtures.BackendAddressPoolName2: {
					{
						Hostname: tests.OtherHost,
					},
				},
			}
			Expect(actual).To(Equal(expected))
		})
	})

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

		It("should be able to prune the managed pools from the lists of all pools", func() {
			blacklisted, notBlacklisted := GetExistingBlacklistedPools(prohibitedTargets, brownfieldContext)

			Expect(len(blacklisted)).To(Equal(2))
			Expect(blacklisted).To(ContainElement(pool1))
			Expect(blacklisted).To(ContainElement(pool2))
			Expect(blacklisted).ToNot(ContainElement(pool3))

			Expect(len(notBlacklisted)).To(Equal(1))
			Expect(notBlacklisted).ToNot(ContainElement(pool1))
			Expect(notBlacklisted).ToNot(ContainElement(pool2))
			Expect(notBlacklisted).To(ContainElement(pool3))
		})
	})
})
