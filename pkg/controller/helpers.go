// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/golang/glog"

	agpoolv1beta1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewaybackendpool/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// Keys that act as cache-busters should be removed from the JSON stored in cache.
var keysToDeleteForCache = []string{
	"etag",
	"tags", // In the Tags of App Gwy we store the timestamp of the most recent update.
	"provisioningState",
	"resourceGuid",
	"location",
}

func (c *AppGwIngressController) updateCache(appGw *n.ApplicationGateway) {
	jsonConfig, err := appGw.MarshalJSON()
	if err != nil {
		glog.Error("Could not marshal App Gwy to update cache; Wiping cache.", err)
		c.configCache = nil
		return
	}
	var sanitized []byte
	if sanitized, err = deleteKeyFromJSON(jsonConfig, keysToDeleteForCache...); err != nil {
		// Ran into an error; Wipe the existing cache
		glog.Error("Failed stripping ETag key from App Gwy config. Wiping cache.", err)
		c.configCache = nil
		return
	}
	*c.configCache = sanitized
}

// configIsSame compares the newly created App Gwy configuration with a cache to determine whether anything has changed.
func (c *AppGwIngressController) configIsSame(appGw *n.ApplicationGateway) bool {
	if c.configCache == nil {
		return false
	}

	sanitizedInput := resetBackReference(appGw)
	jsonConfig, err := sanitizedInput.MarshalJSON()
	if err != nil {
		glog.Error("Could not marshal App Gwy to compare w/ cache; Will not use cache.", err)
		return false
	}
	// The JSON stored in the cache and the newly marshaled JSON will have different ETags even if configs are the same.
	// We need to strip ETags from all nested structures in order to have a fair comparison.
	var sanitized []byte
	if sanitized, err = deleteKeyFromJSON(jsonConfig, keysToDeleteForCache...); err != nil {
		// Ran into an error; Don't use cache; Refresh cache w/ new JSON
		glog.Error("Failed stripping ETag key from App Gwy config. Will not use cache.", err)
		return false
	}

	glog.V(9).Info("input state = ", string(sanitized))
	glog.V(9).Info("cached state = ", string(*c.configCache))

	// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
	return c.configCache != nil && bytes.Compare(*c.configCache, sanitized) == 0
}

// resetBackReference removes the back references in app gateway.
// Currently, This is an issue in redirectConfigurations
func resetBackReference(appGw *n.ApplicationGateway) *n.ApplicationGateway {
	if appGw != nil && appGw.ApplicationGatewayPropertiesFormat != nil {
		if appGw.RedirectConfigurations != nil {
			for idx := range *appGw.RedirectConfigurations {
				(*appGw.RedirectConfigurations)[idx].RequestRoutingRules = nil
				(*appGw.RedirectConfigurations)[idx].URLPathMaps = nil
				(*appGw.RedirectConfigurations)[idx].PathRules = nil
			}
		}
	}
	return appGw
}

func dumpSanitizedJSON(appGw *n.ApplicationGateway, logToFile bool, overwritePrefix *string) ([]byte, error) {
	jsonConfig, err := appGw.MarshalJSON()
	if err != nil {
		return nil, err
	}

	prefix := "-- App Gwy config --"
	if overwritePrefix != nil {
		prefix = *overwritePrefix
	}

	// Remove sensitive data from the JSON config to be logged
	keysToDelete := []string{
		"sslCertificates",
	}
	var sanitized []byte
	if sanitized, err = deleteKeyFromJSON(jsonConfig, keysToDelete...); err != nil {
		return nil, err
	}

	prettyJSON, err := utils.PrettyJSON(sanitized, prefix)

	if logToFile {
		fileName := fmt.Sprintf("app-gateway-config-%d.json", time.Now().UnixNano())
		if filePath, err := utils.SaveToFile(fileName, prettyJSON); err != nil {
			glog.Error("Could not log to file: ", filePath, err)
		}
	}

	return prettyJSON, err
}

func (c *AppGwIngressController) isApplicationGatewayMutable(appGw *n.ApplicationGateway) bool {
	return appGw.OperationalState == "Running" || appGw.OperationalState == "Starting"
}

func isMap(v interface{}) bool {
	return v != nil && reflect.ValueOf(v).Type().Kind() == reflect.Map
}

func isSlice(v interface{}) bool {
	return v != nil && reflect.ValueOf(v).Type().Kind() == reflect.Slice
}

// deleteKey recursively deletes the given key from the map. This is NOT cap sensitive.
func deleteKey(m *map[string]interface{}, keyToDelete string) {
	// Recursively search for the given keyToDelete
	for k, v := range *m {
		if strings.ToLower(k) == strings.ToLower(keyToDelete) {
			delete(*m, k)
			continue
		}

		// Recurse into other maps we find
		if isMap(v) {
			subMap := v.(map[string]interface{})
			deleteKey(&subMap, keyToDelete)
			continue
		}

		// JSON blob could have a list for a value; Iterate and  find maps; Delete matching keys
		if isSlice(v) {
			slice := v.([]interface{})
			for idx := range slice {
				// We are not interested in any other element type but maps
				if isMap(slice[idx]) {
					subMap := slice[idx].(map[string]interface{})
					deleteKey(&subMap, keyToDelete)
				}
			}
		}
	}
}

// deleteKeyFromJSON assumes the []byte passed is JSON. It unmarshalls, deletes the given key, and marshalls again.
func deleteKeyFromJSON(jsonWithEtag []byte, keysToDelete ...string) ([]byte, error) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonWithEtag), &m); err != nil {
		glog.Error("Could not unmarshal config App Gwy JSON to delete Etag.", err)
		return nil, err
	}
	for _, keyToDelete := range keysToDelete {
		deleteKey(&m, keyToDelete)
	}
	return json.Marshal(m)
}

// convert a list of backendaddress to ip addresses
func (c *AppGwIngressController) getIPAddresses(addressPools *[]n.ApplicationGatewayBackendAddress) []agpoolv1beta1.BackendAddress {
	if addressPools == nil {
		return []agpoolv1beta1.BackendAddress{}
	}

	var ipAddresses []agpoolv1beta1.BackendAddress
	for _, addressPool := range *addressPools {
		address := agpoolv1beta1.BackendAddress{IPAddress: *addressPool.IPAddress}
		ipAddresses = append(ipAddresses, address)
	}
	return ipAddresses
}

// compare generated BackendAddressPools with cached one
func (c *AppGwIngressController) isBackendAddressPoolsUpdated(generated, existing *[]n.ApplicationGatewayBackendAddressPool) bool {
	if existing == nil && generated == nil {
		return false
	}

	if existing == nil || generated == nil {
		return true
	}

	backendIDtoIPAddressesMapNew := make(map[string][]string)
	backendIDtoIPAddressesMapExisting := make(map[string]bool)

	for _, gbap := range *generated {
		backendIDNew := gbap.ID
		glog.V(9).Infof("New: find backend pool id: [%s]", *backendIDNew)
		ips := make([]string, len(*gbap.BackendAddresses))
		for i, ip := range *gbap.BackendAddresses {
			ips[i] = *ip.IPAddress
		}
		sort.Strings(ips)
		backendIDtoIPAddressesMapNew[*backendIDNew] = ips
	}

	for _, cbap := range *existing {
		backendIDExisting := cbap.ID
		backendIDtoIPAddressesMapExisting[*backendIDExisting] = true
		if ipAddresses, exists := backendIDtoIPAddressesMapNew[*backendIDExisting]; exists {
			glog.V(9).Infof("Existing: find backend pool name: %s", *backendIDExisting)

			ips := make([]string, len(*cbap.BackendAddresses))
			for i, ip := range *cbap.BackendAddresses {
				ips[i] = *ip.IPAddress
			}
			sort.Strings(ips)
			// indicates the backend pool is updated
			if !equal(ipAddresses, ips) {
				glog.V(5).Infof("New: backend address pool ip: %v", ipAddresses)
				glog.V(5).Infof("Existing: backend address pool ip: %v", ips)
				return true
			}
		} else {
			// if any backend pool is deleted
			glog.V(3).Infof("Existing: NOT find backend pool id: [%s]", *backendIDExisting)
			return true
		}
	}

	// if any backend pool is newly added
	for _, gbap := range *generated {
		backendIDNew := gbap.ID
		if _, exists := backendIDtoIPAddressesMapExisting[*backendIDNew]; !exists {
			return true
		}
	}

	return false
}

// compare two string slices
func equal(a, b []string) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
