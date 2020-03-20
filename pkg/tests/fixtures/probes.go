// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	"encoding/base64"
	"fmt"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

const (
	// ProbeName1 is a string constant.
	ProbeName1 = "probe-name-YnllLmNvbQ-L2Zvbw" // fixtures.Host  fixtures.PathFoo

	// ProbeName2 is a string constant.
	ProbeName2 = "probe-name-YnllLmNvbQ-L2Jhcg" // fixtures.Host  fixtures.PathBar

	// ProbeName3 is a string constant.
	ProbeName3 = "probe-name-LS1zb21lLW90aGVyLWhvc3RuYW1lLS0-L2Zvbw" // fixtures.OtherHost  fixtures.PathFoo
)

// GetApplicationGatewayProbe creates a new struct for use in unit tests.
func GetApplicationGatewayProbe(host *string, path *string) n.ApplicationGatewayProbe {
	if host == nil {
		host = to.StringPtr(tests.Host)
	}
	if path == nil {
		path = to.StringPtr("/foo")
	}

	encodedHost := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(*host)), "=")
	encodedPath := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(*path)), "=")
	probeName := to.StringPtr(fmt.Sprintf("probe-name-%s-%s", encodedHost, encodedPath))

	return n.ApplicationGatewayProbe{
		ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
			Protocol: n.HTTPS,
			Host:     host,
			Path:     path,
		},
		Name: probeName,
		ID:   to.StringPtr("abcd"),
	}
}
