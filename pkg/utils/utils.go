// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

// MaxInt64 returns the greater one of the two
func MaxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// MaxInt32 returns the greater one of the two
func MaxInt32(a int32, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// IntsToString converts a list of int to a string with delim as delimiter
func IntsToString(l []int, delim string) string {
	out := make([]string, len(l))
	for i, v := range l {
		out[i] = strconv.Itoa(v)
	}
	return strings.Join(out, delim)
}

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

// SaveToFile saves the content into a fileName - a tool primarily used for debugging purposes.
func SaveToFile(fileName string, content []byte) {
	tempFile, err := ioutil.TempFile("", fileName)
	if err != nil {
		glog.Error(err)
		return
	}
	if _, err := tempFile.Write(content); err != nil {
		glog.Error(err)
		return
	}
	if err := tempFile.Close(); err != nil {
		glog.Error(err)
		return
	}
	glog.Infof("Saved App Gateway config to %s", tempFile.Name())
}
