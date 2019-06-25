// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"testing"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
	"github.com/Azure/go-autorest/autorest/to"
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test the controller")
}

var _ = Describe("configure App Gateway", func() {
	Context("ensure app gwy is tagged", func() {
		agw := &n.ApplicationGateway{}
		version.Version = "a"
		version.GitCommit = "b"
		version.BuildDate = "c"

		addTags(agw)
		It("should have 1 tag", func() {
			expected := map[string]*string{
				managedByK8sIngress: to.StringPtr("a/b/c"),
			}
			Expect(agw.Tags).To(Equal(expected))
			Expect(len(agw.Tags)).To(Equal(1))
		})
	})

})
