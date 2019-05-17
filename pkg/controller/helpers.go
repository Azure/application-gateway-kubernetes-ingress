package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
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

	// Keep a copy of the stripped JSON string
	*cache = stripped

	fmt.Printf(">>>>> cache:  %p -- %s\n\n", cache, *cache)
	fmt.Printf(">>>>> strip:  %p -- %s\n\n", &stripped, stripped)

	return sameAsCache
}

// deleteKey recursively deletes the given key from the map. This is NOT cap sensitive.
func deleteKey(m *map[string]interface{}, keyToDelete string) {
	for k, v := range *m {
		if strings.ToLower(k) == strings.ToLower(keyToDelete) {
			delete(*m, k)
		} else if reflect.ValueOf(v).Type().Kind() == reflect.Map {
			subMap := v.(map[string]interface{})
			deleteKey(&subMap, keyToDelete)
		} else if reflect.ValueOf(v).Type().Kind() == reflect.Slice {
			valuesSlice := v.([]interface{})
			for idx := range valuesSlice {
				if reflect.ValueOf(valuesSlice[idx]).Type().Kind() == reflect.Map {
					subMap := valuesSlice[idx].(map[string]interface{})
					deleteKey(&subMap, keyToDelete)
				}
			}
		}
	}
}

// deleteKeyFromJSON assumes the []byte passed is JSON. It unmarshalls, deletes the given key, and marshalls again.
func deleteKeyFromJSON(jsonWithEtag []byte, keyToDelete string) ([]byte, error) {
	var stripped map[string]interface{}
	if err := json.Unmarshal([]byte(jsonWithEtag), &stripped); err != nil {
		glog.Error("Could not unmarshal config App Gwy JSON to delete Etag.", err)
		return nil, err
	}
	deleteKey(&stripped, keyToDelete)
	return json.Marshal(stripped)
}
