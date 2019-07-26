package fixtures

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func GetAppGateway() n.ApplicationGateway {
	// The order of the lists below is important as we reference these by index in unit tests.
	return n.ApplicationGateway{
		ApplicationGatewayPropertiesFormat: &n.ApplicationGatewayPropertiesFormat{

			RequestRoutingRules: &[]n.ApplicationGatewayRequestRoutingRule{
				*GetDefaultRoutingRule(),
				*GetRequestRoutingRuleBasic(),
				*GetRequestRoutingRulePathBased1(),
				*GetRequestRoutingRulePathBased2(),
			},
			URLPathMaps: &[]n.ApplicationGatewayURLPathMap{
				*GetDefaultURLPathMap(),
				*GetURLPathMap1(),
				*GetURLPathMap2(),
			},

			HTTPListeners: &[]n.ApplicationGatewayHTTPListener{
				*GetDefaultListener(),
				*GetListenerBasic(),
				*GetListenerPathBased1(),
				*GetListenerPathBased2(),
				*GetListenerUnassociated(),
			},

			SslCertificates: &[]n.ApplicationGatewaySslCertificate{
				GetCertificate1(),
				GetCertificate2(),
				GetCertificate3(),
			},

			Probes: &[]n.ApplicationGatewayProbe{
				GetApplicationGatewayProbe(nil, to.StringPtr(PathFoo)), // /foo
				GetApplicationGatewayProbe(nil, to.StringPtr(PathBar)), // /bar
				GetApplicationGatewayProbe(to.StringPtr(tests.OtherHost), nil),
			},

			BackendHTTPSettingsCollection: &[]n.ApplicationGatewayBackendHTTPSettings{
				GetHTTPSettings1(),
				GetHTTPSettings2(),
				GetHTTPSettings3(),
			},
		},
	}
}
