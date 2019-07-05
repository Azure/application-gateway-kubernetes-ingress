// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

const (
	// PathFoo is a URL path.
	PathFoo = "/foo"

	// PathFox is a URL path.
	PathFox = "/fox"

	// PathBar is a URL path.
	PathBar = "/bar"

	// PathBaz is a URL path.
	PathBaz = "/baz"
)

// GetProhibitedTargets creates a new struct for use in unit tests.
func GetProhibitedTargets() []*ptv1.AzureIngressProhibitedTarget {
	return []*ptv1.AzureIngressProhibitedTarget{
		{
			Spec: ptv1.AzureIngressProhibitedTargetSpec{
				IP:       IPAddress1,
				Hostname: tests.Host,
				Port:     443,
				Paths: []string{
					PathFox,
					PathBar,
				},
			},
		},
	}
}
