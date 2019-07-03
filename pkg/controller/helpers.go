// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

var keysToDeleteForCache = []string{
	"etag",
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
	jsonConfig, err := appGw.MarshalJSON()
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
	// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
	return c.configCache != nil && bytes.Compare(*c.configCache, sanitized) == 0
}

func (c *AppGwIngressController) dumpSanitizedJSON(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext) ([]byte, error) {
	jsonConfig, err := appGw.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// Remove sensitive data from the JSON config to be logged
	keysToDelete := []string{
		"sslCertificates",
	}
	var sanitized []byte
	if sanitized, err = deleteKeyFromJSON(jsonConfig, keysToDelete...); err != nil {
		return nil, err
	}

	prettyJSON, err := utils.PrettyJSON(sanitized, "-- App Gwy config --")

	if cbCtx.EnvVariables.EnableSaveConfigToFile == "true" {
		logToFile(prettyJSON)
	}

	return prettyJSON, err
}

func logToFile(prettyJSON []byte) {
	nowNano := time.Now().UnixNano()
	tempFile, err := ioutil.TempFile("", fmt.Sprintf("app-gateway-config-%d.json", nowNano))
	if err != nil {
		glog.Error(err)
		return
	}
	if _, err := tempFile.Write(prettyJSON); err != nil {
		glog.Error(err)
		return
	}
	if err := tempFile.Close(); err != nil {
		glog.Error(err)
		return
	}
	glog.Infof("Saved App Gateway config to %s", tempFile.Name())
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
