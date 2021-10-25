// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("test helpers", func() {

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
		It("should deal with empty cache and store stuff in it", func() {
			c := AppGwIngressController{
				configCache: to.ByteSlicePtr([]byte{}),
			}
			config := n.ApplicationGateway{
				ID: to.StringPtr("something"),
			}
			Expect(c.configIsSame(&config)).To(BeFalse())
			c.updateCache(&config)
			Expect(c.configIsSame(&config)).To(BeTrue())
			Expect(string(*c.configCache)).To(Equal(`{"id":"something"}`))
		})
	})

	Context("ensure appGw works as expected", func() {
		It("should back references", func() {
			appGw := &n.ApplicationGateway{
				ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
					RedirectConfigurations: &[]n.ApplicationGatewayRedirectConfiguration{
						{
							ApplicationGatewayRedirectConfigurationPropertiesFormat: &n.ApplicationGatewayRedirectConfigurationPropertiesFormat{
								RequestRoutingRules: &[]n.SubResource{{ID: to.StringPtr("id")}},
							},
						},
					},
				},
			}

			out := resetBackReference(appGw)
			Expect((*out.RedirectConfigurations)[0].RequestRoutingRules).To(BeNil())
			Expect((*out.RedirectConfigurations)[0].URLPathMaps).To(BeNil())
			Expect((*out.RedirectConfigurations)[0].PathRules).To(BeNil())
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

	Context("ensure isApplicationGatewayMutable works as expected", func() {
		It("should return true as appgw is running", func() {
			c := AppGwIngressController{}
			config := &n.ApplicationGateway{
				ID: to.StringPtr("something"),
				ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
					OperationalState: "Running",
				},
			}
			Expect(c.isApplicationGatewayMutable(config)).To(BeTrue())
		})

		It("should return true as appgw is starting", func() {
			c := AppGwIngressController{}
			config := &n.ApplicationGateway{
				ID: to.StringPtr("something"),
				ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
					OperationalState: "Starting",
				},
			}
			Expect(c.isApplicationGatewayMutable(config)).To(BeTrue())
		})

		It("should return false as appgw is stopped", func() {
			c := AppGwIngressController{}
			config := &n.ApplicationGateway{
				ID: to.StringPtr("something"),
				ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
					OperationalState: "Stopped",
				},
			}
			Expect(c.isApplicationGatewayMutable(config)).To(BeFalse())
		})

		It("should return false as appgw is stopping", func() {
			c := AppGwIngressController{}
			config := &n.ApplicationGateway{
				ID: to.StringPtr("something"),
				ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
					OperationalState: "Stopping",
				},
			}
			Expect(c.isApplicationGatewayMutable(config)).To(BeFalse())
		})

		It("should return false for valid running state but incorrect casing", func() {
			c := AppGwIngressController{}
			config := &n.ApplicationGateway{
				ID: to.StringPtr("something"),
				ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{
					OperationalState: "running",
				},
			}
			Expect(c.isApplicationGatewayMutable(config)).To(BeFalse())
		})
	})
})
