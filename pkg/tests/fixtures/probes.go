// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

func GetApplicationGatewayProbe(host *string, path *string) n.ApplicationGatewayProbe {
	if host == nil {
		host = to.StringPtr(tests.Host)
	}
	if path == nil {
		path = to.StringPtr("/foo")
	}
	return n.ApplicationGatewayProbe{
		ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
			Protocol: n.HTTPS,
			Host:     host,
			Path:     path,
		},
		Name: to.StringPtr("probe-name"),
		ID:   to.StringPtr("abcd"),
	}
}
