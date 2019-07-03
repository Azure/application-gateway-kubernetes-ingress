// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"
)

func TestAppgw(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Brownfield Deployment Tests")
}

var _ = Describe("test TargetBlacklist/TargetWhitelist health probes", func() {

	Context("Test normalizing  permit/prohibit URL paths", func() {
		actual := NormalizePath("*//*hello/**/*//")
		It("should have exactly 1 record", func() {
			Expect(actual).To(Equal("*//*hello"))
		})
	})

	Context("test GetManagedTargetList", func() {
		actual := GetManagedTargetList(fixtures.GetManagedTargets())
		It("Should have produced correct Managed Targets list", func() {
			Expect(len(*actual)).To(Equal(3))
			{
				expected := Target{
					Hostname: tests.Host,
					Port:     443,
					Path:     to.StringPtr("/foo"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
			{
				expected := Target{
					Hostname: tests.Host,
					Port:     443,
					Path:     to.StringPtr("/bar"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
			{
				expected := Target{
					Hostname: tests.Host,
					Port:     443,
					Path:     to.StringPtr("/baz"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
		})
	})

	Context("test GetProhibitedTargetList", func() {
		actual := GetProhibitedTargetList(fixtures.GetProhibitedTargets())
		It("should have produced correct Prohibited Targets list", func() {
			Expect(len(*actual)).To(Equal(2))
			{
				expected := Target{
					Hostname: tests.Host,
					Port:     443,
					Path:     to.StringPtr("/fox"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
			{
				expected := Target{
					Hostname: tests.Host,
					Port:     443,
					Path:     to.StringPtr("/bar"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
		})
	})

	Context("Test IsIn", func() {
		t1 := Target{
			Hostname: tests.Host,
			Port:     443,
			Path:     to.StringPtr("/bar"),
		}

		t2 := Target{
			Hostname: tests.Host,
			Port:     9898,
			Path:     to.StringPtr("/xyz"),
		}

		targetList := []Target{
			{Hostname: tests.Host,
				Port: 443,
				Path: to.StringPtr("/foo"),
			},
			{Hostname: tests.Host,
				Port: 443,
				Path: to.StringPtr("/bar"),
			},
		}

		It("Should be able to find a new Target in an existing list of Targets", func() {
			Expect(t1.IsIn(&targetList)).To(BeTrue())
			Expect(t2.IsIn(&targetList)).To(BeFalse())
		})
	})
})
