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

func TestParseBoolTrue(t *testing.T) {
	key := "key"
	value := "true"
	ingress.Annotations[key] = value
	parsedVal, err := parseBool(&ingress, key)
	if !parsedVal || err != nil {
		t.Error(fmt.Sprintf("parseBool is expected to return true since %s = %s", key, value))
	}
}

func TestParseBoolFalse(t *testing.T) {
	key := "key"
	value := "false"
	ingress.Annotations[key] = value
	parsedVal, err := parseBool(&ingress, key)
	if parsedVal || err != nil {
		t.Error(fmt.Sprintf("parseBool is expected to return false since %s = %s", key, value))
	}
}

func TestParseBoolInvalid(t *testing.T) {
	key := "key"
	value := "nope"
	ingress.Annotations[key] = value
	parsedVal, err := parseBool(&ingress, key)
	if !errors.IsInvalidContent(err) || parsedVal {
		t.Error(fmt.Sprintf("parseBool is expected to return false since %s = %s", key, value))
	}
}

func TestParseBoolMissingKey(t *testing.T) {
	key := "key"
	delete(ingress.Annotations, key)
	_, err := parseBool(&ingress, key)
	if !errors.IsMissingAnnotations(err) {
		t.Error(fmt.Sprintf("key is expected to return false since there is no %s annotation", key))
	}
}

func TestParseInt32(t *testing.T) {
	key := "key"
	value := "20"
	ingress.Annotations[key] = value
	parsedVal, err := parseInt32(&ingress, key)
	if err != nil && string(parsedVal) != value {
		t.Error(fmt.Sprintf("key retuned %d since %s = %s", parsedVal, key, value))
	}
}

func TestParseInt32Invalid(t *testing.T) {
	key := "key"
	value := "20asd"
	ingress.Annotations[key] = value
	parsedVal, err := parseInt32(&ingress, key)
	if errors.IsInvalidContent(err) {
		t.Error(fmt.Sprintf("key retuned %d since %s = %s", parsedVal, key, value))
	}
}

func TestParseInt32MissingKey(t *testing.T) {
	key := "key"
	delete(ingress.Annotations, key)
	_, err := parseInt32(&ingress, key)
	if !errors.IsMissingAnnotations(err) {
		t.Error(fmt.Sprintf("key is expected to return false since there is no %s annotation", key))
	}
}

func TestParseString(t *testing.T) {
	key := "key"
	value := "/path"
	ingress.Annotations[key] = value
	parsedVal, _ := parseString(&ingress, key)
	if parsedVal != value {
		t.Error(fmt.Sprintf("parseString retuned %s since %s = %s", parsedVal, key, value))
	}
}

func TestParseStringMissingKey(t *testing.T) {
	key := "key"
	delete(ingress.Annotations, key)
	_, err := parseString(&ingress, key)
	if !errors.IsMissingAnnotations(err) {
		t.Error(fmt.Sprintf("key is expected to return false since there is no %s annotation", key))
	}
}
