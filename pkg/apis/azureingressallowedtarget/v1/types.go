// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureIngressAllowedTarget is the targets AGIC is allowed to mutate
type AzureIngressAllowedTarget struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AzureIngressAllowedTargetSpec `json:"spec"`
}

// AzureIngressAllowedTargetSpec defines a list of uniquely identifiable targets for which the AGIC is allowed to mutate config.
type AzureIngressAllowedTargetSpec struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// IP address of the allowed target; Could be the public or private address attached to the Application Gateway
	IP string `json:"ip"`

	// Hostname of the allowed target
	Hostname string `json:"hostname"`

	// +optional
	// Port number of the allowed target
	Port int32 `json:"port,omitempty"`

	// +optional
	// Paths is a list of URL paths, for which the Ingress Controller is allowed from mutating Application Gateway configuration; Must begin with a / and end with /*
	Paths []string `json:"paths,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureIngressAllowedTargetList is the list of allowed targets
type AzureIngressAllowedTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureIngressAllowedTarget `json:"items"`
}
