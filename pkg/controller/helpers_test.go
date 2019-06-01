// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test the helpers")
}

var _ = Describe("configure App Gateway", func() {

	Context("ensure deleteKeyFromJSON works as expected", func() {
		jsonWithEtag := []byte(`{
            "etag":"W/\"d3aa9ec8-fb2a-40fb-ab2c-4ff2902fa11d\"",
            "id":"/subscriptions/xxx",
            "other": {"ETAG":123, "keepThis": 98, "andTHIS": "xyz"},
            "Etag":"delete this"
            }
        `)
		jsonWithoutEtag := []byte(`{"id":"/subscriptions/xxx","other":{"andTHIS":"xyz","keepThis":98}}`)
		It("should have stripped etag", func() {
			Expect(deleteKeyFromJSON(jsonWithEtag, "etag")).To(Equal(jsonWithoutEtag))
		})
	})

	Context("ensure deleteKey works as expected", func() {
		m := map[string]interface{}{
			"deleteThisKey": "value3453451",
			"key2":          "value2",
			"nested": map[string]interface{}{
				"DELETETHISKEY": "value1123123",
				"key2":          "value2",
			},
			"deleteTHISKEY": map[string]interface{}{
				"key3": "ok",
			},
			"list": []interface{}{
				map[string]interface{}{
					"delETETHISKEY": "value1123123",
					"key2":          "value2",
				},
			},
		}
		expected := map[string]interface{}{
			"key2": "value2",
			"nested": map[string]interface{}{
				"key2": "value2",
			},
			"list": []interface{}{
				map[string]interface{}{
					"key2": "value2",
				},
			},
		}
		It("should have stripped etag ignoring capitalization", func() {
			deleteKey(&m, "deleteThiSKEY")
			Expect(m).To(Equal(expected))
		})
	})

	Context("ensure configIsSame works as expected", func() {
		It("should deal with nil cache and store stuff in it", func() {
			client := network.ApplicationGateway{
				Name: to.StringPtr("something"),
			}

			c := AppGwIngressController{}

			Expect(c.configIsSame(&client)).To(BeFalse())
			c.updateCache(&client)
			Expect(c.configIsSame(&client)).To(BeTrue())
			Expect(string(*c.configCache)).To(Equal(`{"name":"something"}`))
		})
	})

	Context("ensure isMap works as expected", func() {
		It("should deal with nil values", func() {
			Expect(isMap(nil)).To(BeFalse())
		})
		It("should return true when passed a map", func() {
			Expect(isMap(make(map[string]interface{}))).To(BeTrue())
		})
		It("should return false when passed a slice", func() {
			Expect(isMap(make([]string, 100))).To(BeFalse())
		})
	})

	Context("ensure isSlice works as expected", func() {
		It("should deal with nil values", func() {
			Expect(isSlice(nil)).To(BeFalse())
		})
		It("should return true when passed a slice", func() {
			Expect(isSlice(make([]string, 100))).To(BeTrue())
		})
		It("should return false when passed a map", func() {
			Expect(isSlice(make(map[string]interface{}))).To(BeFalse())
		})
	})
})
