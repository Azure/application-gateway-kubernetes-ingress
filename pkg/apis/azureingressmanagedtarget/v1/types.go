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

// AzureIngressManagedTarget is the targets AGIC is allowed to mutate
type AzureIngressManagedTarget struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AzureIngressManagedTargetSpec `json:"spec"`
}

// AzureIngressManagedTargetSpec defines a list of uniquely identifiable targets for which the AGIC is explicitly allowed to mutate config.
type AzureIngressManagedTargetSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// IP address of the managed target; Could be the public or private address attached to the Application Gateway
	IP string `json:"ip"`

	// +optional
	// Hostname of the managed target
	Hostname string `json:"hostname,omitempty"`

	// +optional
	// Port number of the managed target
	Port int32 `json:"port,omitempty"`

	// +optional
	// Paths is a list of URL paths, for which the Ingress Controller is managed from mutating Application Gateway configuration; Must begin with a / and end with /*
	Paths []string `json:"paths,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureIngressManagedTargetList is the list of managed targets
type AzureIngressManagedTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureIngressManagedTarget `json:"items"`
}
