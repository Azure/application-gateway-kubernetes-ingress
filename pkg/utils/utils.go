// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/golang/glog"
)

// GetResourceKey generates the key in k8s format for a given resource
func GetResourceKey(namespace, name string) string {
	return fmt.Sprintf("%v/%v", namespace, name)
}

// PrettyJSON Unmarshals and Marshall again with Indent so it is human readable
func PrettyJSON(js []byte, prefix string) ([]byte, error) {
	var jsonObj interface{}
	_ = json.Unmarshal(js, &jsonObj)
	return json.MarshalIndent(jsonObj, prefix, "    ")
}

// GetLastChunkOfSlashed splits a string by slash and returns the last chunk.
func GetLastChunkOfSlashed(s string) string {
	split := strings.Split(s, "/")
	return split[len(split)-1]
}

// SaveToFile saves the content into a file named "fileName" - a tool primarily used for debugging purposes.
func SaveToFile(fileName string, content []byte) (string, error) {
	tempFile, err := ioutil.TempFile("", fileName)
	if err != nil {
		glog.Error(err)
		return tempFile.Name(), err
	}
	if _, err := tempFile.Write(content); err != nil {
		glog.Error(err)
		return tempFile.Name(), err
	}
	if err := tempFile.Close(); err != nil {
		glog.Error(err)
		return tempFile.Name(), err
	}
	glog.Infof("Saved App Gateway config to %s", tempFile.Name())
	return tempFile.Name(), nil
}
