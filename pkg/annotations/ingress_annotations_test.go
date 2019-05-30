// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package annotations

import (
	"fmt"
	"testing"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/errors"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ingress = v1beta1.Ingress{
	ObjectMeta: v1.ObjectMeta{
		Annotations: map[string]string{},
	},
}

const (
	NoError = "Expected to return %s and no error. Returned %v and %v."
	Error   = "Expected to return error %s. Returned %v and %v."
)

func TestParseBoolTrue(t *testing.T) {
	key := "key"
	value := "true"
	ingress.Annotations[key] = value
	parsedVal, err := parseBool(&ingress, key)
	if !parsedVal || err != nil {
		t.Error(fmt.Sprintf(NoError, value, parsedVal, err))
	}
}

func TestParseBoolFalse(t *testing.T) {
	key := "key"
	value := "false"
	ingress.Annotations[key] = value
	parsedVal, err := parseBool(&ingress, key)
	if parsedVal || err != nil {
		t.Error(fmt.Sprintf(NoError, value, parsedVal, err))
	}
}

func TestParseBoolInvalid(t *testing.T) {
	key := "key"
	value := "nope"
	ingress.Annotations[key] = value
	parsedVal, err := parseBool(&ingress, key)
	if !errors.IsInvalidContent(err) {
		t.Error(fmt.Sprintf(Error, err, parsedVal, err))
	}
}

func TestParseBoolMissingKey(t *testing.T) {
	key := "key"
	delete(ingress.Annotations, key)
	parsedVal, err := parseBool(&ingress, key)
	if !errors.IsMissingAnnotations(err) || parsedVal {
		t.Error(fmt.Sprintf(Error, errors.ErrMissingAnnotations, parsedVal, err))
	}
}

func TestParseInt32(t *testing.T) {
	key := "key"
	value := "20"
	ingress.Annotations[key] = value
	parsedVal, err := parseInt32(&ingress, key)
	if err != nil || fmt.Sprint(parsedVal) != value {
		t.Error(fmt.Sprintf(NoError, value, parsedVal, err))
	}
}

func TestParseInt32Invalid(t *testing.T) {
	key := "key"
	value := "20asd"
	ingress.Annotations[key] = value
	parsedVal, err := parseInt32(&ingress, key)
	if errors.IsInvalidContent(err) {
		t.Error(fmt.Sprintf(Error, err, parsedVal, err))
	}
}

func TestParseInt32MissingKey(t *testing.T) {
	key := "key"
	delete(ingress.Annotations, key)
	parsedVal, err := parseInt32(&ingress, key)
	if !errors.IsMissingAnnotations(err) || parsedVal != 0 {
		t.Error(fmt.Sprintf(Error, errors.ErrMissingAnnotations, parsedVal, err))
	}
}

func TestParseString(t *testing.T) {
	key := "key"
	value := "/path"
	ingress.Annotations[key] = value
	parsedVal, err := parseString(&ingress, key)
	if parsedVal != value || err != nil {
		t.Error(fmt.Sprintf(NoError, value, parsedVal, err))
	}
}

func TestParseStringMissingKey(t *testing.T) {
	key := "key"
	delete(ingress.Annotations, key)
	parsedVal, err := parseString(&ingress, key)
	if !errors.IsMissingAnnotations(err) {
		t.Error(fmt.Sprintf(Error, errors.ErrMissingAnnotations, parsedVal, err))
	}
}
