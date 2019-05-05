// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package utils

import (
	"fmt"
	"testing"
)

func TestToSlice(t *testing.T) {
	set := NewUnorderedSet()
	set.Insert("one")
	set.Insert("two")
	set.Insert("one")
	expected := []string{"one", "two"}
	actual := set.ToSlice()
	if len(actual) != 2 {
		t.Error(fmt.Sprintf("Expected length to be 2; It is %d", len(actual)))
	}

	if actual[0] != expected[0] && actual[1] != expected[1] {
		t.Error(fmt.Sprintf("\nExpected: %+v\nActually: %+v\n", expected, actual))
	}
}
