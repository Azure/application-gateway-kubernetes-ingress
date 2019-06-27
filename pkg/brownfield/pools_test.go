// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("test blacklist/whitelist backend pools", func() {

	listeners := []*n.ApplicationGatewayHTTPListener{
		fixtures.GetListener1(),
		fixtures.GetListener2(),
	}

	routingRules := []n.ApplicationGatewayRequestRoutingRule{
		*fixtures.GetRequestRoutingRulePathBased(),
		*fixtures.GetRequestRoutingRuleBasic(),
	}

	paths := []n.ApplicationGatewayURLPathMap{
		*fixtures.GeURLPathMap(),
	}

	expected := map[string][]Target{
		fixtures.BackendAddressPoolName1: {
			{
				Hostname: tests.Host,
				Port:     443,
				Path:     to.StringPtr(fixtures.PathFoo),
			},

			{
				Hostname: tests.Host,
				Port:     443,
				Path:     to.StringPtr(fixtures.PathBar),
			},

			{
				Hostname: tests.Host,
				Port:     443,
				Path:     to.StringPtr(fixtures.PathBaz),
			},
		},

		fixtures.BackendAddressPoolName2: {
			{
				Hostname: tests.OtherHost,
				Port:     80,
			},
		},
	}

	Context("Test normalizing  permit/prohibit URL paths", func() {

		actual := GetPoolToTargetMapping(listeners, routingRules, paths)

		It("should have created map of pool name to list of targets", func() {
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Test MergePools()", func() {

		pool1 := fixtures.GetBackendPool1()
		pool2 := fixtures.GetBackendPool2()

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

	Context("Test GetManagedPools()", func() {

		It("should be able to merge lists of pools", func() {
			pools := []n.ApplicationGatewayBackendAddressPool{
				fixtures.GetBackendPool1(),
				fixtures.GetBackendPool2(),
			}
			managedTargets := fixtures.GetManagedTargets()
			prohibitedTargets := fixtures.GetProhibitedTargets()

			// !! Action !!
			actual := GetManagedPools(pools, managedTargets, prohibitedTargets, listeners, routingRules, paths)

			Expect(len(actual)).To(Equal(2))
			Expect(actual).To(ContainElement(fixtures.GetBackendPool1()))
			Expect(actual).To(ContainElement(fixtures.GetBackendPool2()))
		})
	})

	Context("Test PruneManagedPools()", func() {

		It("should be able to merge lists of pools", func() {
			pools := []n.ApplicationGatewayBackendAddressPool{
				fixtures.GetBackendPool1(),
				fixtures.GetBackendPool2(),
				fixtures.GetBackendPool3(),
			}
			managedTargets := fixtures.GetManagedTargets()
			prohibitedTargets := fixtures.GetProhibitedTargets()

			// !! Action !!
			actual := PruneManagedPools(pools, managedTargets, prohibitedTargets, listeners, routingRules, paths)

			Expect(len(actual)).To(Equal(1))
			Expect(actual).To(ContainElement(fixtures.GetBackendPool3()))
		})
	})

})
