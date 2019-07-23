package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

// LookupIPConfigurationByType gets the public or private address depenging upon privateIP parameter.
func LookupIPConfigurationByType(frontendIPConfigurations *[]n.ApplicationGatewayFrontendIPConfiguration, privateIP bool) *n.ApplicationGatewayFrontendIPConfiguration {
	for _, ip := range *frontendIPConfigurations {
		if ip.ApplicationGatewayFrontendIPConfigurationPropertiesFormat != nil &&
			((privateIP && ip.PrivateIPAddress != nil) ||
				(!privateIP && ip.PublicIPAddress != nil)) {
			return &ip
		}
	}

	return nil
}
