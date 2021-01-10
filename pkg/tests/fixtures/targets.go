// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	atv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressallowedtarget/v1"
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
		{
			Spec: ptv1.AzureIngressProhibitedTargetSpec{
				Hostname: tests.WildcardHost1,
			},
		},
	}
}

// GetAzureIngressAllowedTargets creates a new struct for use in unit tests.
func GetAzureIngressAllowedTargets() []*atv1.AzureIngressAllowedTarget {
	return []*atv1.AzureIngressAllowedTarget{
		{
			Spec: atv1.AzureIngressAllowedTargetSpec{
				Hostname: tests.Host,
				Paths: []string{
					PathFox,
					PathBar,
				},
			},
		},
		{
			Spec: atv1.AzureIngressAllowedTargetSpec{
				Hostname: tests.OtherHost,
			},
		},
		{
			Spec: atv1.AzureIngressAllowedTargetSpec{
				Paths: []string{
					PathForbidden,
				},
			},
		},
		{
			Spec: atv1.AzureIngressAllowedTargetSpec{
				Hostname: tests.WildcardHost1,
			},
		},
	}
}
