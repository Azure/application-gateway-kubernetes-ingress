// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package utils

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

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
		klog.Error(err)
		return tempFile.Name(), err
	}
	if _, err := tempFile.Write(content); err != nil {
		klog.Error(err)
		return tempFile.Name(), err
	}
	if err := tempFile.Close(); err != nil {
		klog.Error(err)
		return tempFile.Name(), err
	}
	klog.Infof("Saved App Gateway config to %s", tempFile.Name())
	return tempFile.Name(), nil
}

// GetHashCode generates hashcode of given type
func GetHashCode(i interface{}) string {
	jsonBytes, err := json.Marshal(i)
	if err != nil {
		klog.Errorf("Failed MD5 hashing %+v", i)
		return ""
	}
	return fmt.Sprintf("%x", md5.Sum(jsonBytes))
}

// RandStringRunes generates n length random string
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// RemoveDuplicates returns a copy of a slice with duplicates removed
func RemoveDuplicateStrings(list []string) []string {
	if list == nil {
		return list
	}

	result := []string{}
	// use a map to enforce uniqueness
	dupeChecker := make(map[string]interface{})
	for _, val := range list {
		if _, ok := dupeChecker[val]; !ok {
			result = append(result, val)
			dupeChecker[val] = nil
		}
	}

	return result
}
