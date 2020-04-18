// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// AzureApplicationGatewayBackendPool is the resource AGIC is watching on for any backend address change
type AzureApplicationGatewayBackendPool struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec AzureApplicationGatewayBackendPoolSpec `json:"spec"`
}

// AzureApplicationGatewayBackendPoolSpec defines a list of backend pool addresses
type AzureApplicationGatewayBackendPoolSpec struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// BackendAddressPools include a list of Application Gateway backend pools
	BackendAddressPools []BackendAddressPool `json:"backendAddressPools,omitempty"`
}

// BackendAddressPool defines a backend pool name and a list of backend addresses
type BackendAddressPool struct {
	Name             string           `json:"name,omitempty"`
	BackendAddresses []BackendAddress `json:"backendAddresses,omitempty"`
}

// BackendAddress includes IP address
type BackendAddress struct {
	IPAddress string `json:"ipAddress,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureApplicationGatewayBackendPoolList is the list of backend pool
type AzureApplicationGatewayBackendPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureApplicationGatewayBackendPool `json:"items"`
}
