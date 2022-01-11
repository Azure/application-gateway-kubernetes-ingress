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

// AzureApplicationGatewayHeaderRewrite is the resource AGIC is watching on for any rewrite rules
type AzureApplicationGatewayHeaderRewrite struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AzureApplicationGatewayHeaderRewriteSpec `json:"spec"`
}

// AzureApplicationGatewayBackendPoolSpec defines a list of backend pool addresses
type AzureApplicationGatewayHeaderRewriteSpec struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Rewrite rule name
	RewriteRule string `json:"rewriteRule,omitempty"`

	// Action include a list of rewrite actions
	Actions []Action `json:"action,omitempty"`
}

// Action defines the header crud operation
type Action struct {
	RewriteType string `json:"rewriteType,omitempty"`
	ActionType  string `json:"actionType,omitempty"`
	HeaderName  string `json:"headerName,omitempty"`
	Headervalue string `json:"headerValue,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureApplicationGatewayBackendPoolList is the list of backend pool
type AzureApplicationGatewayHeaderRewriteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureApplicationGatewayHeaderRewrite `json:"items"`
}
