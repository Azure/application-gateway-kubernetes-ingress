package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

// LookupIPConfigurationByType gets the public or private address depending upon privateIP parameter.
func LookupIPConfigurationByType(frontendIPConfigurations *[]n.ApplicationGatewayFrontendIPConfiguration, privateIP bool) *n.ApplicationGatewayFrontendIPConfiguration {
	// If a private frontend is not requested, and there is
	// only 1 frontend present on the gateway then return the
	// frontend whether it is a public or private frontend.
	if !privateIP && len(*frontendIPConfigurations) == 1 {
		return &(*frontendIPConfigurations)[0]
	}

	for _, ip := range *frontendIPConfigurations {
		if ip.ApplicationGatewayFrontendIPConfigurationPropertiesFormat != nil &&
			((privateIP && ip.PrivateIPAddress != nil) ||
				(!privateIP && ip.PublicIPAddress != nil)) {
			return &ip
		}
	}

	return nil
}

// LookupIPConfigurationByID gets by ID.
func LookupIPConfigurationByID(frontendIPConfigurations *[]n.ApplicationGatewayFrontendIPConfiguration, ID *string) *n.ApplicationGatewayFrontendIPConfiguration {
	for _, ip := range *frontendIPConfigurations {
		if *ip.ID == *ID {
			return &ip
		}
	}
	return nil
}

// IsPrivateIPConfiguration returns true if frontendIPConfiguration uses private IP
func IsPrivateIPConfiguration(frontendIPConfiguration *n.ApplicationGatewayFrontendIPConfiguration) bool {
	if frontendIPConfiguration.ApplicationGatewayFrontendIPConfigurationPropertiesFormat != nil && frontendIPConfiguration.PrivateIPAddress != nil {
		return true
	}
	return false
}
