// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"
)

// configIsSame compares the newly created App Gwy configuration with a cache to determine whether anything has changed.
func configIsSame(appGw *network.ApplicationGateway, cache *[]byte) bool {
	jsonConfig, err := appGw.MarshalJSON()
	if err != nil {
		glog.Error("Could not marshal App Gwy to compare w/ cache; Will not use cache.", err)
		return false
	}

	// The JSON stored in the cache and the newly marshalled JSON will have different ETags even if configs are the same.
	// We need to strip ETags from all nested structures in order to maintain cache.
	var stripped []byte
	if stripped, err = deleteKeyFromJSON(jsonConfig, "etag"); err != nil {
		// Ran into an error; Don't use cache; Refresh cache w/ new JSON
		glog.Error("Failed stripping ETag key from App Gwy config. Will not use cache.", err)
		return false
	}
	// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
	sameAsCache := bytes.Compare(*cache, stripped) == 0

	if !sameAsCache {
		// Keep a copy of the stripped JSON string
		*cache = stripped
	}

	return sameAsCache
}

func isMap(v interface{}) bool {
	return reflect.ValueOf(v).Type().Kind() == reflect.Map
}

func isSlice(v interface{}) bool {
	return reflect.ValueOf(v).Type().Kind() == reflect.Slice
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
func deleteKeyFromJSON(jsonWithEtag []byte, keyToDelete string) ([]byte, error) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonWithEtag), &m); err != nil {
		glog.Error("Could not unmarshal config App Gwy JSON to delete Etag.", err)
		return nil, err
	}
	deleteKey(&m, keyToDelete)
	return json.Marshal(m)
}
