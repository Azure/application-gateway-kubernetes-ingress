// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build unittest

package annotations

import (
	"testing"
)

func TestIsMissingAnnotations(t *testing.T) {
	if !IsMissingAnnotations(ErrMissingAnnotations) {
		t.Error("expected true")
	}
}

func TestInvalidContent(t *testing.T) {
	if IsInvalidContent(ErrMissingAnnotations) {
		t.Error("expected false")
	}
	err := NewInvalidAnnotationContent("demo", "")
	if !IsInvalidContent(err) {
		t.Error("expected true")
	}
	if IsInvalidContent(nil) {
		t.Error("expected false")
	}
}
