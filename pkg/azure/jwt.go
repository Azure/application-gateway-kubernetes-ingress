// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

// EntraJWTValidationConfig is the ARM representation of an Application Gateway Entra JWT validation config.
type EntraJWTValidationConfig struct {
	Name       *string                                  `json:"name,omitempty"`
	ID         *string                                  `json:"id,omitempty"`
	Etag       *string                                  `json:"etag,omitempty"`
	Properties *EntraJWTValidationConfigPropertiesFormat `json:"properties,omitempty"`
}

// EntraJWTValidationConfigPropertiesFormat holds Entra JWT validation properties.
type EntraJWTValidationConfigPropertiesFormat struct {
	ClientID                 *string  `json:"clientId,omitempty"`
	TenantID                 *string  `json:"tenantId,omitempty"`
	Audiences                *[]string `json:"audiences,omitempty"`
	UnauthorizedRequestAction *string  `json:"unAuthorizedRequestAction,omitempty"`
}

// JWTMergePayload carries Entra JWT configs and per-rule bindings for gateway update.
type JWTMergePayload struct {
	Configs      []EntraJWTValidationConfig
	RuleBindings map[string]string // request routing rule name -> JWT config name
}

// EmptyJWTMerge returns an empty payload.
func EmptyJWTMerge() *JWTMergePayload {
	return &JWTMergePayload{
		RuleBindings: map[string]string{},
	}
}
