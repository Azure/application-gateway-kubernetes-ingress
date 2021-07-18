// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package v1

import (
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureApplicationGatewayLoadDistributionPolicy is the resource AGIC is watching on for any backend address change
type AzureApplicationGatewayLoadDistributionPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec AzureApplicationGatewayLoadDistributionPolicySpec `json:"spec"`
}

// AzureApplicationGatewayLoadDistributionPolicySpec defines a list of backend pool addresses
type AzureApplicationGatewayLoadDistributionPolicySpec struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	//Targets include a list of backend targets
	Targets []Backend `json:"Targets,omitempty"`
}

// Backend defines a backend service and its load distribution policy parameters
type Backend struct {
	Role    string                   `json:"role,omitempty"`
	Weight  int                      `json:"weight,omitempty"`
	Service v1.IngressServiceBackend `json:"service,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureApplicationGatewayLoadDistributionPolicyList is the list of LDP
type AzureApplicationGatewayLoadDistributionPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureApplicationGatewayLoadDistributionPolicy `json:"items"`
}
