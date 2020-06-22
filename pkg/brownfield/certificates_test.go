// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests/fixtures"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
)

var _ = Describe("Test MergeCerts", func() {
	Context("Test MergeCerts()", func() {
		It("should function as expected", func() {
			bucket1 := []n.ApplicationGatewaySslCertificate{
				fixtures.GetCertificate1(),
				fixtures.GetCertificate2(),
			}
			bucket2 := []n.ApplicationGatewaySslCertificate{
				fixtures.GetCertificate1(),
				fixtures.GetCertificate3(),
			}
			actual := MergeCerts(bucket1, bucket2)
			Expect(actual).To(ContainElement(fixtures.GetCertificate1()))
			Expect(actual).To(ContainElement(fixtures.GetCertificate2()))
			Expect(actual).To(ContainElement(fixtures.GetCertificate3()))
		})
	})
})
