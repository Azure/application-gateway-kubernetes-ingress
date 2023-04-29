package appgw

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
)

type FrontendType string

const (
	// FrontendTypePublic is a public IP address
	FrontendTypePublic FrontendType = "Public"

	// FrontendTypePrivate is a private IP address
	FrontendTypePrivate FrontendType = "Private"
)

// LookupIPConfigurationByType gets the public or private address depending upon privateIP parameter.
func LookupIPConfigurationByType(frontendIPConfigurations *[]n.ApplicationGatewayFrontendIPConfiguration, frontendType FrontendType) *n.ApplicationGatewayFrontendIPConfiguration {
	for _, ip := range *frontendIPConfigurations {
		switch frontendType {
		case FrontendTypePublic:
			if ip.PublicIPAddress != nil {
				return &ip
			}
		case FrontendTypePrivate:
			if ip.PrivateIPAddress != nil {
				return &ip
			}
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

// DetermineFrontendType determines whether frontend is public or private.
func DetermineFrontendType(frontendIPConfiguration *n.ApplicationGatewayFrontendIPConfiguration) FrontendType {
	if frontendIPConfiguration.PrivateIPAddress != nil {
		return FrontendTypePrivate
	}
	return FrontendTypePublic
}
