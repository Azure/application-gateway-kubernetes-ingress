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
	// Name of the rewrite rule
	Name string `json:"name,omitempty"`

	// RuleSequence determines the order of execution of a particular rule in a RewriteRuleSet
	RuleSequence int `json:"ruleSequence,omitempty"`

	// Actions contain the set of actions to be done as part of the rewrite rule
	Actions Actions `json:"actions,omitempty"`

	// Conditions is a list of conditions based on which the action set execution will be evaluated
	Conditions []Condition `json:"conditions,omitempty"`
}

// Condition includes IgnoreCase, Negate, Variable and Pattern
type Condition struct {
	// IgnoreCase set to true will do a case in-sensitive comparison.
	IgnoreCase bool `json:"ignoreCase,omitempty"`

	// Negate set as true will check the negation of the given condition
	Negate bool `json:"negate,omitempty"`

	// Variable is the condition parameter
	Variable string `json:"variable,omitempty"`

	// Pattern, either fixed string or regular expression, that evaluates the truthfulness of the condition
	Pattern string `json:"pattern,omitempty"`
}

// Actions includes RequestHeaderConfigurations, ResponseHeaderConfigurations and UrlConfiguration
type Actions struct {
	// RequestHeaderConfigurations is a set of Request Header Actions in the Action Set
	RequestHeaderConfigurations []HeaderConfiguration `json:"requestHeaderConfigurations,omitempty"`

	// ResponseHeaderConfigurations is a set of Response Header Actions in the Action Set
	ResponseHeaderConfigurations []HeaderConfiguration `json:"responseHeaderConfigurations,omitempty"`

	// UrlConfiguration is the URL Configuration Action in the Action
	UrlConfiguration UrlConfiguration `json:"urlConfiguration,omitempty"`
}

// HeaderConfiguration includes ActionType, HeaderName and HeaderValue
type HeaderConfiguration struct {
	// ActionType is the type of manipulation that should be performed on the header. Should be either 'set' or 'delete'
	ActionType string `json:"actionType,omitempty"`

	// 	HeaderName is the name of the header to manipulate
	HeaderName string `json:"headerName,omitempty"`

	// 	HeaderValue is the value of the header. Empty in case ActionType is 'delete'
	HeaderValue string `json:"headerValue,omitempty"`
}

// UrlConfiguration includes ModifiedPath, ModifiedQueryString and Reroute
type UrlConfiguration struct {
	// ModifiedPath is the URL path for URL rewrite. Empty string means no path will be updated
	ModifiedPath string `json:"modifiedPath,omitempty"`

	// ModifiedQueryString is the query string for url rewrite. Empty string means no query string will be updated
	ModifiedQueryString string `json:"modifiedQueryString,omitempty"`

	// Reroute set as true will re-evaluate the url path map provided in using modified path. Default value is false
	Reroute bool `json:"reroute,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureApplicationGatewayRewriteList is the list of backend pool
type AzureApplicationGatewayRewriteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureApplicationGatewayRewrite `json:"items"`
}
