// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"testing"
)

func TestGenerateSSLRedirectConfigurationName(t *testing.T) {
	namespace := "nmspc"
	ingress := "ingrs"
	actual := generateSSLRedirectConfigurationName(namespace, ingress)
	expected := "k8s-ag-ingress-nmspc-ingrs-sslr"
	if actual != expected {
		t.Error(fmt.Sprintf("\nExpected %s\nActually %s", expected, actual))
	}
}

