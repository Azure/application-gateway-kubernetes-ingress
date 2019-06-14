// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AzureIngressProhibitedTarget is the targets AGIC is not allowed to mutate
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AzureIngressProhibitedTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AzureIngressProhibitedTargetSpec `json:"spec"`
}

// AzureIngressProhibitedTargetSpec defines a list of uniquely identifiable targets for which the AGIC is not allowed to mutate config.
type AzureIngressProhibitedTargetSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// IP address of the prohibited target; Could be the public or private address attached to the Application Gateway
	IP string `json:"ip"`
	// Hostname of the prohibited target
	Hostname string `json:"hostname,omitempty"`
	// Port number of the prohibited target
	Port int32 `json:"port,omitempty"`
	// Paths is a list of URL paths, for which the Ingress Controller is prohibited from mutating Application Gateway configuration; Must begin with a / and end with /*
	Paths []string `json:"paths,omitempty"`
}

// AzureIngressProhibitedTargetList is the list of prohibited targets
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AzureIngressProhibitedTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureIngressProhibitedTarget `json:"items"`
}
