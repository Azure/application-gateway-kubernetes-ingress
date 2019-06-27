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

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("test blacklist/whitelist backend pools", func() {

	listeners := []*n.ApplicationGatewayHTTPListener{
		fixtures.GetListenerWithPathBasedRules(),
		fixtures.GetListenerWithBasicRules(),
	}

	requestRoutingRules := []n.ApplicationGatewayRequestRoutingRule{
		*fixtures.GetRequestRoutingRulePathBased(),
		*fixtures.GetRequestRoutingRuleBasic(),
	}

	paths := []n.ApplicationGatewayURLPathMap{
		*fixtures.GeURLPathMap(),
	}

	expected := map[string]Target{
		"BackendAddressPool-1": {
			Hostname: "xxx.yyy.zzz",
			Port:     443,
			Path:     to.StringPtr("/bar"),
		},
		"BackendAddressPool-2": {
			Hostname: "aaa.bbb.ccc",
			Port:     80,
		},
	}

	Context("Test normalizing  permit/prohibit URL paths", func() {

		actual := GetPoolToTargetMapping(listeners, requestRoutingRules, paths)

		It("should have exactly 1 record", func() {
			Expect(actual).To(Equal(expected))
		})
	})
})
