package fixtures

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

const (
	// DefaultPortName is a string constant.
	DefaultPortName = "fp-80"
)

// GetDefaultPort creates a struct used for unit testing.
func GetDefaultPort() n.ApplicationGatewayFrontendPort {
	return n.ApplicationGatewayFrontendPort{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(DefaultPortName),
		ID:   to.StringPtr("/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontendPorts/fp-80"),
		ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
			Port: to.Int32Ptr(int32(80)),
		},
	}
}

// GetPort creates a struct used for unit testing.
func GetPort(portNo int32) n.ApplicationGatewayFrontendPort {
	return n.ApplicationGatewayFrontendPort{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(fmt.Sprintf("fp-%d", portNo)),
		ID:   to.StringPtr(fmt.Sprintf("/subscriptions/--subscription--/resourceGroups/--resource-group--/providers/Microsoft.Network/applicationGateways/--app-gw-name--/frontendPorts/fp-%d", portNo)),
		ApplicationGatewayFrontendPortPropertiesFormat: &n.ApplicationGatewayFrontendPortPropertiesFormat{
			Port: to.Int32Ptr(portNo),
		},
	}
}
