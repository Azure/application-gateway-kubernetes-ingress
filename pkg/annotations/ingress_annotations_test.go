// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package annotations

import (
	"fmt"
	"testing"

	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ingress = v1beta1.Ingress{
	ObjectMeta: v1.ObjectMeta{
		Annotations: map[string]string{},
	},
}

func TestIsSslRedirectYes(t *testing.T) {
	redirectKey := "true"
	ingress.Annotations[SslRedirectKey] = redirectKey
	if !IsSslRedirect(&ingress) {
		t.Error(fmt.Sprintf("IsSslRedirect is expected to return true since %s = %s", SslRedirectKey, redirectKey))
	}
}

func TestIsSslRedirectNo(t *testing.T) {
	redirectKey := "nope"
	ingress.Annotations[SslRedirectKey] = redirectKey
	if IsSslRedirect(&ingress) {
		t.Error(fmt.Sprintf("IsSslRedirect is expected to return false since %s = %s", SslRedirectKey, redirectKey))
	}
}
func TestIsSslRedirectMissingKey(t *testing.T) {
	delete(ingress.Annotations, SslRedirectKey)
	if IsSslRedirect(&ingress) {
		t.Error(fmt.Sprintf("IsSslRedirect is expected to return false since there is no %s annotation", SslRedirectKey))
	}
}

