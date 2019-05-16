// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package utils

import (
	"fmt"
	"os"
	"regexp"
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

// GetEnv is an augmentation of os.Getenv, providing it with a default value.
func GetEnv(environmentVariable, defaultValue string, validator *regexp.Regexp) string {
	if value, ok := os.LookupEnv(environmentVariable); ok {
		if validator == nil {
			return value
		}
		if validator.MatchString(value) {
			return value
		} else {
			glog.Errorf("Environment variable %s contains a value which does not pass validation filter; Using default value: %s", environmentVariable, defaultValue)
		}
	}
	return defaultValue
}
