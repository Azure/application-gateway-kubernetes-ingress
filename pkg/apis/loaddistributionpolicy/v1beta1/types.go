// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package v1beta1

import (
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadDistributionPolicy is the resource AGIC is watching on for any backend address change
type LoadDistributionPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec LoadDistributionPolicySpec `json:"spec"`
}

// LoadDistributionPolicySpec defines a list of backend pool addresses
type LoadDistributionPolicySpec struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	//Targets include a list of backend targets
	Targets []Target `json:"targets,omitempty"`
}

// Target defines a backend service and its load distribution parameters
type Target struct {
	Role    string  `json:"role,omitempty"`
	Weight  int     `json:"weight,omitempty"`
	Backend Backend `json:"backend,omitempty"`
}

// Backend defines a backend service
type Backend struct {
	Service *v1.IngressServiceBackend `json:"service,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadDistributionPolicyList is the list of LDP
type LoadDistributionPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []LoadDistributionPolicy `json:"items"`
}
