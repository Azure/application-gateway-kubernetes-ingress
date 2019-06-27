package fixtures

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func GetListenerWithPathBasedRules() *network.ApplicationGatewayHTTPListener {
	return &network.ApplicationGatewayHTTPListener{
		Name: to.StringPtr("HTTPListener-PathBased"),
		ApplicationGatewayHTTPListenerPropertiesFormat: &network.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration:     &network.SubResource{ID: to.StringPtr("")},
			FrontendPort:                &network.SubResource{ID: to.StringPtr("")},
			Protocol:                    network.HTTPS,
			HostName:                    to.StringPtr("xxx.yyy.zzz"),
			SslCertificate:              &network.SubResource{ID: to.StringPtr("")},
			RequireServerNameIndication: to.BoolPtr(true),
		},
	}
}

func GetListenerWithBasicRules() *network.ApplicationGatewayHTTPListener {
	return &network.ApplicationGatewayHTTPListener{
		Name: to.StringPtr("HTTPListener-Basic"),
		ApplicationGatewayHTTPListenerPropertiesFormat: &network.ApplicationGatewayHTTPListenerPropertiesFormat{
			FrontendIPConfiguration:     &network.SubResource{ID: to.StringPtr("")},
			FrontendPort:                &network.SubResource{ID: to.StringPtr("")},
			Protocol:                    network.HTTP,
			HostName:                    to.StringPtr("aaa.bbb.ccc"),
			SslCertificate:              &network.SubResource{ID: to.StringPtr("")},
			RequireServerNameIndication: to.BoolPtr(true),
		},
	}
}
