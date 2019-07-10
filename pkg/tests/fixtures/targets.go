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

	// PathForbidden is a URL path.
	PathForbidden = "/forbidden-path"
)

// GetAzureIngressProhibitedTargets creates a new struct for use in unit tests.
func GetAzureIngressProhibitedTargets() []*ptv1.AzureIngressProhibitedTarget {
	return []*ptv1.AzureIngressProhibitedTarget{
		{
			Spec: ptv1.AzureIngressProhibitedTargetSpec{
				Hostname: tests.Host,
				Paths: []string{
					PathFox,
					PathBar,
				},
			},
		},
		{
			Spec: ptv1.AzureIngressProhibitedTargetSpec{
				Hostname: tests.OtherHost,
			},
		},
		{
			Spec: ptv1.AzureIngressProhibitedTargetSpec{
				Paths: []string{
					PathForbidden,
				},
			},
		},
	}
}
