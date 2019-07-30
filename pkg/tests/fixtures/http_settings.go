package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	DefaultBackendHTTPSettingsName = "DefaultBackendHTTPSettings"

	// BackendHTTPSettingsName1 is a string constant.
	BackendHTTPSettingsName1 = "BackendHTTPSettings-1"

	// BackendHTTPSettingsName2 is a string constant.
	BackendHTTPSettingsName2 = "BackendHTTPSettings-2"

	// BackendHTTPSettingsName3 is a string constant.
	BackendHTTPSettingsName3 = "BackendHTTPSettings-3"
)

func GetHTTPSettings1() n.ApplicationGatewayBackendHTTPSettings {
	return n.ApplicationGatewayBackendHTTPSettings{
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Probe: &n.SubResource{ID: to.StringPtr("/x/y/z/" + ProbeName1)},
		},
		Name: to.StringPtr(BackendHTTPSettingsName1),
	}
}

func GetHTTPSettings2() n.ApplicationGatewayBackendHTTPSettings {
	return n.ApplicationGatewayBackendHTTPSettings{
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Probe: &n.SubResource{ID: to.StringPtr("/x/y/z/" + ProbeName2)},
		},
		Name: to.StringPtr(BackendHTTPSettingsName2),
	}
}

func GetHTTPSettings3() n.ApplicationGatewayBackendHTTPSettings {
	return n.ApplicationGatewayBackendHTTPSettings{
		ApplicationGatewayBackendHTTPSettingsPropertiesFormat: &n.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Probe: &n.SubResource{ID: to.StringPtr("/x/y/z/" + ProbeName3)},
		},
		Name: to.StringPtr(BackendHTTPSettingsName3),
	}
}
