// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

var _ = Describe("Test blacklisting targets", func() {

	expected := Target{
		Hostname: tests.Host,
	}

	Context("test GetTargetBlacklist", func() {
		blacklist := GetTargetBlacklist(fixtures.GetAzureIngressProhibitedTargets())
		It("should have produced correct Prohibited Targets list", func() {
			Expect(len(*blacklist)).To(Equal(5))

			// Targets /fox and /bar are in the blacklist
			for _, path := range []string{fixtures.PathFox, fixtures.PathBar} {
				expected.Path = TargetPath(path)
				Expect(*blacklist).To(ContainElement(expected))
			}
			expected.Path = fixtures.PathFoo
			Expect(*blacklist).ToNot(ContainElement(expected))
		})
	})

	Context("Test IsBlacklisted", func() {
		targetInBlacklist := Target{
			Hostname: tests.Host,
			Path:     fixtures.PathBar,
		}

		targetInBlacklistNoHost := Target{
			Hostname: tests.Host,
			Path:     "/xyz",
		}

		targetNoPaths := Target{
			Hostname: "other-host-no-paths",
		}

		targetNonExistentPath := Target{
			Hostname: tests.Host,
			Path:     fixtures.PathBar + "Non-Existent-Path",
		}

		targetNoHost := Target{
			Path: fixtures.PathBar,
		}

		blacklist := []Target{
			{
				Hostname: tests.Host,
				Path:     fixtures.PathFoo,
			},
			{
				Hostname: tests.Host,
				Path:     fixtures.PathBar,
			},
			{
				Hostname: "other-host-no-paths",
			},
			{
				Path: "/xyz",
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

	Context("test TargetPath.contains(TargetPath)", func() {
		It("TargetPath.contains(TargetPath) should work correctly", func() {
			Expect(TargetPath("/*").contains("/blah")).To(BeTrue())
			Expect(TargetPath("/*").contains("/")).To(BeTrue())
			Expect(TargetPath("/*").contains("")).To(BeTrue())
			Expect(TargetPath("/*").contains("/*")).To(BeTrue())

			Expect(TargetPath("*").contains("/blah")).To(BeTrue())
			Expect(TargetPath("*").contains("/")).To(BeTrue())
			Expect(TargetPath("*").contains("")).To(BeTrue())
			Expect(TargetPath("*").contains("/*")).To(BeTrue())

			Expect(TargetPath("/").contains("/blah")).To(BeFalse())
			Expect(TargetPath("/blah").contains("/blah")).To(BeTrue())
			Expect(TargetPath("/").contains("/")).To(BeTrue())
			Expect(TargetPath("/").contains("")).To(BeFalse())
			Expect(TargetPath("/").contains("/*")).To(BeFalse())

			Expect(TargetPath("/x/*").contains("/blah")).To(BeFalse())
			Expect(TargetPath("/x/*").contains("/")).To(BeFalse())
			Expect(TargetPath("/x/*").contains("")).To(BeFalse())
			Expect(TargetPath("/x/*").contains("/*")).To(BeFalse())

			Expect(TargetPath("/x/*").contains("/x")).To(BeTrue())
			Expect(TargetPath("/x/*").contains("/X")).To(BeTrue())
			Expect(TargetPath("/X/*").contains("/x")).To(BeTrue())
			Expect(TargetPath("/x/*").contains("/x/")).To(BeTrue())
			Expect(TargetPath("/x/*").contains("/x/*")).To(BeTrue())

			Expect(TargetPath("/abCD/xyZ/1/*").contains("/AbcD/Xyz/*")).To(BeFalse())
			Expect(TargetPath("/AbcD/Xyz/*").contains("/abCD/xyZ/1/*")).To(BeTrue())
		})
	})

	Context("Test getProhibitedHostNames()", func() {
		er := ExistingResources{
			ProhibitedTargets: []*v1.AzureIngressProhibitedTarget{
				{
					Spec: v1.AzureIngressProhibitedTargetSpec{
						Hostname: tests.Host,
					},
				},
				{
					Spec: v1.AzureIngressProhibitedTargetSpec{
						Paths: []string{
							"/a",
							"/b",
						},
					},
				},
			},
		}
		It("should create a list of prohibited HostNames", func() {
			prohibitedHostNames := er.getProhibitedHostNames()
			Expect(len(prohibitedHostNames)).To(Equal(1))
			expected := map[string]interface{}{
				tests.Host: nil,
			}
			Expect(er.getProhibitedHostNames()).To(Equal(expected))
		})
	})

	Context("Test hostname regex matching", func() {
		It("should match hostnames using regex patterns", func() {
			// Test regex pattern matching for subdomains
			blacklistWithRegex := []Target{
				{
					Hostname:              "",
					compiledHostnameRegex: compileTestRegex("^.*\\.example\\.com$"),
				},
			}

			Expect(Target{Hostname: "api.example.com"}.IsBlacklisted(&blacklistWithRegex)).To(BeTrue())
			Expect(Target{Hostname: "www.example.com"}.IsBlacklisted(&blacklistWithRegex)).To(BeTrue())
			Expect(Target{Hostname: "dev.staging.example.com"}.IsBlacklisted(&blacklistWithRegex)).To(BeTrue())
			Expect(Target{Hostname: "example.com"}.IsBlacklisted(&blacklistWithRegex)).To(BeFalse())
			Expect(Target{Hostname: "notexample.com"}.IsBlacklisted(&blacklistWithRegex)).To(BeFalse())
		})

		It("should be case-insensitive for regex patterns", func() {
			blacklistWithRegex := []Target{
				{
					Hostname:              "",
					compiledHostnameRegex: compileTestRegex("^api\\.example\\.com$"),
				},
			}

			Expect(Target{Hostname: "api.example.com"}.IsBlacklisted(&blacklistWithRegex)).To(BeTrue())
			Expect(Target{Hostname: "API.EXAMPLE.COM"}.IsBlacklisted(&blacklistWithRegex)).To(BeTrue())
			Expect(Target{Hostname: "Api.Example.Com"}.IsBlacklisted(&blacklistWithRegex)).To(BeTrue())
		})

		It("should prefer regex over exact match when regex is present", func() {
			blacklistWithRegex := []Target{
				{
					Hostname:              "exact.example.com",
					compiledHostnameRegex: compileTestRegex("^.*\\.staging\\.com$"),
				},
			}

			// Should use regex, not exact match
			Expect(Target{Hostname: "api.staging.com"}.IsBlacklisted(&blacklistWithRegex)).To(BeTrue())
			Expect(Target{Hostname: "exact.example.com"}.IsBlacklisted(&blacklistWithRegex)).To(BeFalse())
		})

		It("should fall back to exact match when no regex is present", func() {
			blacklistExact := []Target{
				{
					Hostname: "exact.example.com",
				},
			}

			Expect(Target{Hostname: "exact.example.com"}.IsBlacklisted(&blacklistExact)).To(BeTrue())
			Expect(Target{Hostname: "EXACT.EXAMPLE.COM"}.IsBlacklisted(&blacklistExact)).To(BeTrue()) // case-insensitive
			Expect(Target{Hostname: "api.example.com"}.IsBlacklisted(&blacklistExact)).To(BeFalse())
		})
	})

	Context("Test GetTargetBlacklist with regex", func() {
		It("should compile valid regex patterns", func() {
			prohibitedTargets := []*v1.AzureIngressProhibitedTarget{
				{
					Spec: v1.AzureIngressProhibitedTargetSpec{
						HostnameRegex: "^.*\\.staging\\.com$",
						Paths:         []string{"/api/*"},
					},
				},
			}

			blacklist := GetTargetBlacklist(prohibitedTargets)
			Expect(len(*blacklist)).To(Equal(1))
			Expect((*blacklist)[0].compiledHostnameRegex).ToNot(BeNil())
			Expect((*blacklist)[0].Hostname).To(Equal("")) // Hostname should be cleared when using regex
		})

		It("should skip invalid regex patterns and log error", func() {
			prohibitedTargets := []*v1.AzureIngressProhibitedTarget{
				{
					Spec: v1.AzureIngressProhibitedTargetSpec{
						HostnameRegex: "[invalid(regex",
					},
				},
				{
					Spec: v1.AzureIngressProhibitedTargetSpec{
						Hostname: "valid.example.com",
					},
				},
			}

			blacklist := GetTargetBlacklist(prohibitedTargets)
			// Invalid regex should be skipped, only valid hostname should remain
			Expect(len(*blacklist)).To(Equal(1))
			Expect((*blacklist)[0].Hostname).To(Equal("valid.example.com"))
			Expect((*blacklist)[0].compiledHostnameRegex).To(BeNil())
		})

		It("should prioritize hostnameRegex over hostname field", func() {
			prohibitedTargets := []*v1.AzureIngressProhibitedTarget{
				{
					Spec: v1.AzureIngressProhibitedTargetSpec{
						Hostname:      "exact.example.com",
						HostnameRegex: "^.*\\.example\\.com$",
					},
				},
			}

			blacklist := GetTargetBlacklist(prohibitedTargets)
			Expect(len(*blacklist)).To(Equal(1))
			Expect((*blacklist)[0].compiledHostnameRegex).ToNot(BeNil())
			Expect((*blacklist)[0].Hostname).To(Equal("")) // Should be cleared when regex is used

			// Verify regex matching works
			Expect(Target{Hostname: "api.example.com"}.IsBlacklisted(blacklist)).To(BeTrue())
			Expect(Target{Hostname: "exact.example.com"}.IsBlacklisted(blacklist)).To(BeTrue())
		})
	})
})

// Helper function to compile regex patterns for testing
func compileTestRegex(pattern string) *regexp.Regexp {
	compiled, _ := regexp.Compile("(?i)" + pattern)
	return compiled
}
