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
		Port:     443,
	}

	Context("Test normalizing permit/prohibit URL paths", func() {
		actual := NormalizePath("*//*heLLo/**/*//")
		It("should have exactly 1 record", func() {
			Expect(actual).To(Equal("*//*hello"))
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

	Context("Test IsBlacklisted", func() {
		targetInBlacklist := Target{
			Hostname: tests.Host,
			Port:     443,
			Path:     to.StringPtr(fixtures.PathBar),
		}

		targetPartialPathInBlacklist := Target{
			Hostname: tests.Host,
			Port:     443,
			Path:     to.StringPtr("/path/with/"),
		}

		targetPartialPathNotInBlacklist := Target{
			Hostname: tests.Host,
			Port:     443,
			Path:     to.StringPtr("/path/with/XXX/"),
		}

		targetNotInList := Target{
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

		targetNoHost := Target{
			Port: 443,
			Path: to.StringPtr(fixtures.PathBar),
		}

		blacklist := []Target{
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
				Path:     to.StringPtr("/path/with/high/specificity/*"),
			},
			{
				Hostname: "other-host-no-paths",
				Port:     123,
			},
		}

		It("Should be able to find a new Target in an existing list of Targets", func() {
			// Blacklisted targets
			Expect(targetInBlacklist.IsBlacklisted(&blacklist)).To(BeTrue())
			Expect(targetPartialPathInBlacklist.IsBlacklisted(&blacklist)).To(BeTrue())

			// Non-blacklisted targets
			Expect(targetPartialPathNotInBlacklist.IsBlacklisted(&blacklist)).To(BeFalse())
			Expect(targetNotInList.IsBlacklisted(&blacklist)).To(BeFalse())
			Expect(targetNoPort.IsBlacklisted(&blacklist)).To(BeTrue())
			Expect(targetNoPaths.IsBlacklisted(&blacklist)).To(BeTrue())
			Expect(targetNonExistentPath.IsBlacklisted(&blacklist)).To(BeFalse())
			Expect(targetNoHost.IsBlacklisted(&blacklist)).To(BeFalse())
		})
	})

	Context("", func() {

		It("Should be able to determine that a needle is in a haystack", func() {
			needle := "/a/b/c/d/e/f/g/*"
			haystack := "/a/b/*"
			Expect(pathsOverlap(needle, haystack)).To(BeTrue())
		})

		It("Should be to determine that a needle is covering the entire haystack", func() {
			needle := "/a/b/c"
			haystack := "/a/b/c/d/e/f/g/*"
			Expect(pathsOverlap(needle, haystack)).To(BeTrue())
		})

		It("Should be able to determine that the needle is not in the haystack", func() {
			needle := "/a/b/c"
			haystack := "/x/y/z/*"
			Expect(pathsOverlap(needle, haystack)).To(BeFalse())
		})

	})

})
