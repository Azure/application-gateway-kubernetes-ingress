// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/klog/v2"
)

// Application Gateway API version that includes entraJWTValidationConfigs.
const appGwJWTAPIVersion = "2025-03-01"

type gatewayJWTEnvelope struct {
	Properties *struct {
		EntraJWTValidationConfigs *[]EntraJWTValidationConfig `json:"entraJWTValidationConfigs"`
	} `json:"properties"`
}

func (az *azClient) gatewayURL() string {
	return fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/applicationGateways/%s",
		az.appGatewaysClient.BaseURI,
		string(az.subscriptionID),
		string(az.resourceGroupName),
		string(az.appGwName),
	)
}

func (az *azClient) authorizedDo(req *http.Request) (*http.Response, error) {
	preparer := autorest.CreatePreparer(az.appGatewaysClient.Authorizer.WithAuthorization())
	prepared, err := preparer.Prepare(req)
	if err != nil {
		return nil, err
	}
	sender := az.appGatewaysClient.Sender
	if sender == nil {
		sender = autorest.CreateSender()
	}
	return sender.Do(prepared)
}

// extractEntraJWTConfigs GETs the gateway with a JWT-capable API version and returns existing configs.
func (az *azClient) extractEntraJWTConfigs() ([]EntraJWTValidationConfig, error) {
	url := fmt.Sprintf("%s?api-version=%s", az.gatewayURL(), appGwJWTAPIVersion)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(az.ctx)
	req.Header.Set("Accept", "application/json")

	resp, err := az.authorizedDo(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET application gateway for JWT configs failed: status=%d body=%s", resp.StatusCode, truncate(body, 512))
	}

	var envelope gatewayJWTEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope.Properties == nil || envelope.Properties.EntraJWTValidationConfigs == nil {
		return nil, nil
	}
	return *envelope.Properties.EntraJWTValidationConfigs, nil
}

// updateGatewayWithJWT PUTs the track1 gateway body merged with JWT configs/bindings using a JWT-capable API version.
func (az *azClient) updateGatewayWithJWT(appGw *n.ApplicationGateway, jwt *JWTMergePayload) error {
	raw, err := appGw.MarshalJSON()
	if err != nil {
		return err
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}

	props, _ := doc["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
		doc["properties"] = props
	}

	if jwt == nil {
		jwt = EmptyJWTMerge()
	}
	if jwt.Configs == nil {
		jwt.Configs = []EntraJWTValidationConfig{}
	}

	configsJSON, err := json.Marshal(jwt.Configs)
	if err != nil {
		return err
	}
	var configsAny interface{}
	if err := json.Unmarshal(configsJSON, &configsAny); err != nil {
		return err
	}
	// Always set the collection (including empty) so portal configs are explicitly preserved or cleared only via merge logic.
	props["entraJWTValidationConfigs"] = configsAny

	if len(jwt.RuleBindings) > 0 {
		rules, ok := props["requestRoutingRules"].([]interface{})
		if !ok {
			klog.Warning("requestRoutingRules missing or unexpected type; skipping Entra JWT rule bindings")
		} else {
			for i, ruleAny := range rules {
				rule, ok := ruleAny.(map[string]interface{})
				if !ok {
					continue
				}
				name, _ := rule["name"].(string)
				cfgName, bound := jwt.RuleBindings[name]
				if !bound || cfgName == "" {
					continue
				}
				ruleProps, _ := rule["properties"].(map[string]interface{})
				if ruleProps == nil {
					ruleProps = map[string]interface{}{}
					rule["properties"] = ruleProps
				}
				ruleProps["entraJWTValidationConfig"] = map[string]interface{}{
					"id": entraJWTValidationConfigResourceID(string(az.subscriptionID), string(az.resourceGroupName), string(az.appGwName), cfgName),
				}
				rules[i] = rule
			}
			props["requestRoutingRules"] = rules
		}
	}

	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s?api-version=%s", az.gatewayURL(), appGwJWTAPIVersion)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req = req.WithContext(az.ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := az.authorizedDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	if resp.StatusCode == http.StatusAccepted {
		return az.waitForAzureAsync(resp)
	}

	return fmt.Errorf("PUT application gateway with JWT configs failed: status=%d body=%s", resp.StatusCode, truncate(body, 1024))
}

func (az *azClient) waitForAzureAsync(resp *http.Response) error {
	future, err := azure.NewFutureFromResponse(resp)
	if err != nil {
		return err
	}
	pt := future.PollingURL()
	if pt != "" {
		klog.V(3).Infof("OperationID='%s'", GetOperationIDFromPollingURL(pt))
	}
	return future.WaitForCompletionRef(az.ctx, az.appGatewaysClient.Client)
}

func entraJWTValidationConfigResourceID(subscriptionID, resourceGroup, appGwName, configName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/applicationGateways/%s/entraJWTValidationConfigs/%s",
		subscriptionID, resourceGroup, appGwName, configName)
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// MergeEntraJWTConfigs upserts helmConfigs over existing by name; configs not in helmConfigs are preserved untouched.
func MergeEntraJWTConfigs(existing []EntraJWTValidationConfig, helmConfigs []EntraJWTValidationConfig) []EntraJWTValidationConfig {
	helmByName := map[string]EntraJWTValidationConfig{}
	for _, cfg := range helmConfigs {
		if cfg.Name == nil {
			continue
		}
		helmByName[*cfg.Name] = cfg
	}

	var out []EntraJWTValidationConfig
	seen := map[string]struct{}{}

	for _, cfg := range existing {
		if cfg.Name == nil {
			continue
		}
		name := *cfg.Name
		if helm, ok := helmByName[name]; ok {
			out = append(out, helm)
			seen[name] = struct{}{}
			continue
		}
		out = append(out, cfg)
		seen[name] = struct{}{}
	}

	for name, cfg := range helmByName {
		if _, ok := seen[name]; ok {
			continue
		}
		out = append(out, cfg)
	}

	return out
}

// NewHelmEntraJWTConfig builds an ARM JWT config from Helm fields.
func NewHelmEntraJWTConfig(subscriptionID, resourceGroup, appGwName, name, tenantID, clientID, unauthorizedAction string, audiences []string) EntraJWTValidationConfig {
	if unauthorizedAction == "" {
		unauthorizedAction = "Deny"
	}
	props := &EntraJWTValidationConfigPropertiesFormat{
		TenantID:                  to.StringPtr(tenantID),
		ClientID:                  to.StringPtr(clientID),
		UnauthorizedRequestAction: to.StringPtr(unauthorizedAction),
	}
	if len(audiences) > 0 {
		aud := append([]string(nil), audiences...)
		props.Audiences = &aud
	}
	id := entraJWTValidationConfigResourceID(subscriptionID, resourceGroup, appGwName, name)
	return EntraJWTValidationConfig{
		Name:       to.StringPtr(name),
		ID:         to.StringPtr(id),
		Etag:       to.StringPtr("*"),
		Properties: props,
	}
}

// ConfigNames returns the set of JWT config names.
func ConfigNames(configs []EntraJWTValidationConfig) map[string]struct{} {
	out := map[string]struct{}{}
	for _, cfg := range configs {
		if cfg.Name != nil && strings.TrimSpace(*cfg.Name) != "" {
			out[*cfg.Name] = struct{}{}
		}
	}
	return out
}
