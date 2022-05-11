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

// AzureApplicationGatewayRewrite is the resource AGIC is watching on for any rewrite rule change
type AzureApplicationGatewayRewrite struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec AzureApplicationGatewayRewriteSpec `json:"spec"`
}

// AzureApplicationGatewayRewriteSpec defines a list of rewrite rules
type AzureApplicationGatewayRewriteSpec struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// RewriteRules include a list of Application Gateway rewrite rules
	RewriteRules []RewriteRule `json:"rewriteRules,omitempty"`
}

// RewriteRule defines a rewrite rule name, rule sequence, a list of conditions and actions
type RewriteRule struct {
	Name         string      `json:"name,omitempty"`
	RuleSequence int         `json:"ruleSequence,omitempty"`
	Actions      Actions     `json:"actions,omitempty"`
	Conditions   []Condition `json:"conditions,omitempty"`
}

// Condition includes IgnoreCase, Negate, Variable and Pattern
type Condition struct {
	IgnoreCase bool   `json:"ignoreCase,omitempty"`
	Negate     bool   `json:"negate,omitempty"`
	Variable   string `json:"variable,omitempty"`
	Pattern    string `json:"pattern,omitempty"`
}

// Actions includes RequestHeaderConfigurations, ResponseHeaderConfigurations and UrlConfiguration
type Actions struct {
	RequestHeaderConfigurations  []RequestHeaderConfiguration  `json:"requestHeaderConfigurations,omitempty"`
	ResponseHeaderConfigurations []ResponseHeaderConfiguration `json:"responseHeaderConfigurations,omitempty"`
	UrlConfiguration             UrlConfiguration              `json:"urlConfiguration,omitempty"`
}

// RequestHeaderConfiguration and ResponseHeaderConfiguration can be same

// RequestHeaderConfiguration includes ActionType, HeaderName and HeaderValue
type RequestHeaderConfiguration struct {
	ActionType  string `json:"actionType,omitempty"`
	HeaderName  string `json:"headerName,omitempty"`
	HeaderValue string `json:"headerValue,omitempty"`
}

// ResponseHeaderConfiguration includes ActionType, HeaderName and HeaderValue
type ResponseHeaderConfiguration struct {
	ActionType  string `json:"actionType,omitempty"`
	HeaderName  string `json:"headerName,omitempty"`
	HeaderValue string `json:"headerValue,omitempty"`
}

// ResponseHeaderConfiguration includes ModifiedPath, ModifiedQueryString and Reroute
type UrlConfiguration struct {
	ModifiedPath        string `json:"modifiedPath,omitempty"`
	ModifiedQueryString string `json:"modifiedQueryString,omitempty"`
	Reroute             bool   `json:"reroute,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureApplicationGatewayRewriteList is the list of backend pool
type AzureApplicationGatewayRewriteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureApplicationGatewayRewrite `json:"items"`
}
