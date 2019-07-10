package fixtures

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	// DefaultPortName is a string constant.
	DefaultPortName = "fp-80"
)

// GetDefaultPort creates a struct used for unit testing.
func GetDefaultPort() n.ApplicationGatewayFrontendPort {
	return n.ApplicationGatewayFrontendPort{
		Name: to.StringPtr(DefaultPortName),
	}
}
