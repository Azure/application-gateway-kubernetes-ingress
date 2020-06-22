// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

const (
	// DefaultHTTPListenerName is a string constant.
	DefaultHTTPListenerName = "fl-80"

	// HTTPListenerNameBasic is a string constant.
	HTTPListenerNameBasic = "HTTPListener-Basic"

	// HTTPListenerPathBased1 is a string constant.
	HTTPListenerPathBased1 = "HTTPListener-PathBased"

	// HTTPListenerPathBased2 is a string constant.
	HTTPListenerPathBased2 = "HTTPListener-PathBased2"

	// HTTPListenerUnassociated is a string constant.
	HTTPListenerUnassociated = "HTTPListener-Unassociated"

	// HTTPListenerWildcard is a string constant.
	HTTPListenerWildcard = "HTTPListener-Wildcard"
)

// GetListenerBasic creates a new struct for use in unit tests.
func GetListenerBasic() *n.ApplicationGatewayHTTPListener {
	return &n.ApplicationGatewayHTTPListener{
		Name: to.StringPtr(HTTPListenerNameBasic),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration:     &n.SubResource{ID: to.StringPtr("")},
			FrontendPort:                &n.SubResource{ID: to.StringPtr("")},
			Protocol:                    n.HTTP,
			HostName:                    to.StringPtr(tests.OtherHost),
			SslCertificate:              &n.SubResource{ID: to.StringPtr(CertificateName1)},
			RequireServerNameIndication: to.BoolPtr(true),
		},
	}
}

// GetDefaultListener creates a new struct for use in unit tests.
func GetDefaultListener() *n.ApplicationGatewayHTTPListener {
	return &n.ApplicationGatewayHTTPListener{
		Name: to.StringPtr(DefaultHTTPListenerName),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration: &n.SubResource{ID: to.StringPtr("/x/y/z/" + DefaultIPName)},
			FrontendPort:            &n.SubResource{ID: to.StringPtr("/x/y/z/" + DefaultPortName)},
			Protocol:                n.HTTP,
		},
	}
}

// GetListenerPathBased1 creates a new struct for use in unit tests.
func GetListenerPathBased1() *n.ApplicationGatewayHTTPListener {
	return &n.ApplicationGatewayHTTPListener{
		Name: to.StringPtr(HTTPListenerPathBased1),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration:     &n.SubResource{ID: to.StringPtr("")},
			FrontendPort:                &n.SubResource{ID: to.StringPtr("")},
			Protocol:                    n.HTTPS,
			HostName:                    to.StringPtr(tests.Host),
			SslCertificate:              &n.SubResource{ID: to.StringPtr(CertificateName2)},
			RequireServerNameIndication: to.BoolPtr(true),
		},
	}
}

// GetListenerPathBased2 creates a new struct for use in unit tests.
func GetListenerPathBased2() *n.ApplicationGatewayHTTPListener {
	return &n.ApplicationGatewayHTTPListener{
		Name: to.StringPtr(HTTPListenerPathBased2),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration:     &n.SubResource{ID: to.StringPtr("")},
			FrontendPort:                &n.SubResource{ID: to.StringPtr("")},
			Protocol:                    n.HTTP,
			HostName:                    to.StringPtr(tests.OtherHost),
			SslCertificate:              &n.SubResource{ID: to.StringPtr(CertificateName3)},
			RequireServerNameIndication: to.BoolPtr(true),
		},
	}
}

// GetListenerUnassociated creates a new listener, which is not associated with routing rules etc.
func GetListenerUnassociated() *n.ApplicationGatewayHTTPListener {
	return &n.ApplicationGatewayHTTPListener{
		Name: to.StringPtr(HTTPListenerUnassociated),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration:     &n.SubResource{ID: to.StringPtr("")},
			FrontendPort:                &n.SubResource{ID: to.StringPtr("")},
			Protocol:                    n.HTTP,
			HostName:                    to.StringPtr(tests.HostUnassociated),
			RequireServerNameIndication: to.BoolPtr(true),
		},
	}
}

// GetListenerWildcard creates a new listener which is associated to a rule and uses wild card HostNames
func GetListenerWildcard() *n.ApplicationGatewayHTTPListener {
	return &n.ApplicationGatewayHTTPListener{
		Name: to.StringPtr(HTTPListenerWildcard),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration:     &n.SubResource{ID: to.StringPtr("")},
			FrontendPort:                &n.SubResource{ID: to.StringPtr("")},
			Protocol:                    n.HTTP,
			HostNames:                   &[]string{tests.WildcardHost1, tests.WildcardHost2},
			RequireServerNameIndication: to.BoolPtr(true),
		},
	}
}
