package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

const (
	Listener1 = "HTTPListener-PathBased"
	Listener2 = "HTTPListener-Basic"
)

func GetListener1() *n.ApplicationGatewayHTTPListener {
	return &n.ApplicationGatewayHTTPListener{
		Name: to.StringPtr(Listener1),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration:     &n.SubResource{ID: to.StringPtr("")},
			FrontendPort:                &n.SubResource{ID: to.StringPtr("")},
			Protocol:                    n.HTTPS,
			HostName:                    to.StringPtr(tests.Host),
			SslCertificate:              &n.SubResource{ID: to.StringPtr("")},
			RequireServerNameIndication: to.BoolPtr(true),
		},
	}
}

func GetListener2() *n.ApplicationGatewayHTTPListener {
	return &n.ApplicationGatewayHTTPListener{
		Name: to.StringPtr(Listener2),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration:     &n.SubResource{ID: to.StringPtr("")},
			FrontendPort:                &n.SubResource{ID: to.StringPtr("")},
			Protocol:                    n.HTTP,
			HostName:                    to.StringPtr(tests.OtherHost),
			SslCertificate:              &n.SubResource{ID: to.StringPtr("")},
			RequireServerNameIndication: to.BoolPtr(true),
		},
	}
}
