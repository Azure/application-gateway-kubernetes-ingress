// Package utils : utilities
package utils

import (
	"strconv"
	"strings"
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
