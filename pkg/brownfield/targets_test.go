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

var _ = Describe("test TargetBlacklist/TargetWhitelist health probes", func() {

	expected := Target{
		Hostname: tests.Host,
		Port:     443,
	}

	Context("Test normalizing permit/prohibit URL paths", func() {
		actual := NormalizePath("*//*hello/**/*//")
		It("should have exactly 1 record", func() {
			Expect(actual).To(Equal("*//*hello"))
		})
	})

	Context("test GetTargetWhitelist", func() {
		whitelist := GetTargetWhitelist(fixtures.GetManagedTargets())
		It("Should have produced correct Managed Targets list", func() {
			Expect(len(*whitelist)).To(Equal(3))
			for _, path := range []string{fixtures.PathFoo, fixtures.PathBar, fixtures.PathBaz} {
				expected.Path = to.StringPtr(path)
				Expect(*whitelist).To(ContainElement(expected))
			}
			expected.Path = to.StringPtr(fixtures.PathFox)
			Expect(*whitelist).ToNot(ContainElement(expected))

		})
	})

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

	Context("Test IsIn", func() {
		t1 := Target{
			Hostname: tests.Host,
			Port:     443,
			Path:     to.StringPtr(fixtures.PathBar),
		}

		t2 := Target{
			Hostname: tests.Host,
			Port:     9898,
			Path:     to.StringPtr("/xyz"),
		}

		targetNoPort := Target{
			Hostname: tests.Host,
			Path:     to.StringPtr(fixtures.PathBar),
		}

		targetNoPaths := Target{
			Hostname: "other-host-no-paths",
			Port:     123,
		}

		targetNonExistentPath := Target{
			Hostname: tests.Host,
			Port:     443,
			Path:     to.StringPtr(fixtures.PathBar + "Non-Existent-Path"),
		}

		targetList := []Target{
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
				Hostname: "other-host-no-paths",
				Port:     123,
			},
		}

		It("Should be able to find a new Target in an existing list of Targets", func() {
			Expect(t1.IsIn(&targetList)).To(BeTrue())
			Expect(t2.IsIn(&targetList)).To(BeFalse())
			Expect(targetNoPort.IsIn(&targetList)).To(BeTrue())
			Expect(targetNoPaths.IsIn(&targetList)).To(BeTrue())
			Expect(targetNonExistentPath.IsIn(&targetList)).To(BeFalse())
		})
	})

})
