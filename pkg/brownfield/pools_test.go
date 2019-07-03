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

var _ = Describe("test TargetBlacklist/TargetWhitelist backend pools", func() {

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

	brownfieldContext := PoolContext{
		Listeners:    listeners,
		RoutingRules: routingRules,
		PathMaps:     paths,
	}

	targetFoo := Target{
		Hostname: tests.Host,
		Port:     443,
		Path:     to.StringPtr(fixtures.PathFoo),
	}

	targets := poolToTargets{
		fixtures.BackendAddressPoolName1: {
			targetFoo,

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

	pool1 := fixtures.GetBackendPool1()
	pool2 := fixtures.GetBackendPool2()
	pool3 := fixtures.GetBackendPool3()

	// Create a list of pools
	pools := []n.ApplicationGatewayBackendAddressPool{
		pool1, // managed
		pool2, // managed
		pool3, // unmanaged / prohibited
	}

	Context("Test normalizing  permit/prohibit URL paths", func() {

		actual := brownfieldContext.getPoolToTargets()

		It("should have created map of pool name to list of targets", func() {
			Expect(actual).To(Equal(targets))
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

	Context("Test GetManagedPools()", func() {

		It("should be able to merge lists of pools", func() {

			managedTargets := fixtures.GetManagedTargets()
			prohibitedTargets := fixtures.GetProhibitedTargets()

			// !! Action !!
			actual := GetManagedPools(pools, managedTargets, prohibitedTargets, brownfieldContext)

			Expect(len(actual)).To(Equal(2))
			Expect(actual).To(ContainElement(pool1))
			Expect(actual).To(ContainElement(pool2))
			Expect(actual).ToNot(ContainElement(pool3))
		})
	})

	Context("Test PruneManagedPools()", func() {

		It("should be able to merge lists of pools", func() {
			managedTargets := fixtures.GetManagedTargets()
			prohibitedTargets := fixtures.GetProhibitedTargets()

			// !! Action !!
			actual := PruneManagedPools(pools, managedTargets, prohibitedTargets, brownfieldContext)

			Expect(len(actual)).To(Equal(1))
			Expect(actual).To(ContainElement(pool3))
		})
	})

	Context("Apply Blacklist", func() {

		It("should filter a list of backend pools based on a Blacklist", func() {
			blacklistedTargets := []Target{targetFoo}

			// !! Action !!
			actual := brownfieldContext.applyBlacklist(pools, &blacklistedTargets)

			Expect(len(pools)).To(Equal(3))
			Expect(len(actual)).To(Equal(2))
			Expect(actual).To(ContainElement(pool1))
			Expect(actual).To(ContainElement(pool2))
			Expect(actual).ToNot(ContainElement(pool3))
		})
	})

	Context("Apply Whitelist", func() {

		It("should filter a list of backend pools based on a Whitelist", func() {
			whitelistedTargets := []Target{targetFoo}

			// !! Action !!
			actual := brownfieldContext.applyWhitelist(pools, &whitelistedTargets)

			Expect(len(pools)).To(Equal(3))
			Expect(len(actual)).To(Equal(1))
			Expect(actual).To(ContainElement(pool1))
			Expect(actual).ToNot(ContainElement(pool2))
			Expect(actual).ToNot(ContainElement(pool3))
		})
	})

})
