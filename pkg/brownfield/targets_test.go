// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test blacklisting targets", func() {

	expected := Target{
		Hostname: tests.Host,
	}

	Context("test GetTargetBlacklist", func() {
		blacklist := GetTargetBlacklist(fixtures.GetProhibitedTargets())
		It("should have produced correct Prohibited Targets list", func() {
			Expect(len(*blacklist)).To(Equal(2))

			// Targets /fox and /bar are in the blacklist
			for _, path := range []string{fixtures.PathFox, fixtures.PathBar} {
				expected.Path = to.StringPtr(path)
				Expect(*blacklist).To(ContainElement(expected))
			}
			expected.Path = to.StringPtr(fixtures.PathFoo)
			Expect(*blacklist).ToNot(ContainElement(expected))
		})
	})

	Context("Test IsBlacklisted", func() {
		targetInBlacklist := Target{
			Hostname: tests.Host,
			Path:     to.StringPtr(fixtures.PathBar),
		}

		targetInBlacklistNoHost := Target{
			Hostname: tests.Host,
			Path:     to.StringPtr("/xyz"),
		}

		targetNoPaths := Target{
			Hostname: "other-host-no-paths",
		}

		targetNonExistentPath := Target{
			Hostname: tests.Host,
			Path:     to.StringPtr(fixtures.PathBar + "Non-Existent-Path"),
		}

		targetNoHost := Target{
			Path: to.StringPtr(fixtures.PathBar),
		}

		blacklist := []Target{
			{
				Hostname: tests.Host,
				Path:     to.StringPtr(fixtures.PathFoo),
			},
			{
				Hostname: tests.Host,
				Path:     to.StringPtr(fixtures.PathBar),
			},
			{
				Hostname: "other-host-no-paths",
			},
			{
				Path: to.StringPtr("/xyz"),
			},
		}

		It("Should be able to find a new Target in an existing list of Targets", func() {
			// Blacklisted targets
			Expect(targetInBlacklist.IsBlacklisted(&blacklist)).To(BeTrue())
			Expect(targetInBlacklistNoHost.IsBlacklisted(&blacklist)).To(BeTrue())

			// Non-blacklisted targets
			Expect(targetNoPaths.IsBlacklisted(&blacklist)).To(BeTrue())
			Expect(targetNonExistentPath.IsBlacklisted(&blacklist)).To(BeFalse())
			Expect(targetNoHost.IsBlacklisted(&blacklist)).To(BeFalse())
		})
	})

})
