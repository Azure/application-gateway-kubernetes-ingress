// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// NewAppGwyConfigFixture creates a new struct for testing.
func NewAppGwyConfigFixture() *n.ApplicationGatewayPropertiesFormat {
	feIPConfigs := []n.ApplicationGatewayFrontendIPConfiguration{
		{
			// Public IP
			Name: to.StringPtr("xx3"),
			Etag: to.StringPtr("xx2"),
			Type: to.StringPtr("xx1"),
			ID:   to.StringPtr(tests.PublicIPID),
			ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &n.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
				PrivateIPAddress: nil,
				PublicIPAddress: &n.SubResource{
					ID: to.StringPtr("xyz"),
				},
			},
		},
		{
			// Private IP
			Name: to.StringPtr("yy3"),
			Etag: to.StringPtr("yy2"),
			Type: to.StringPtr("yy1"),
			ID:   to.StringPtr(tests.PrivateIPID),
			ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &n.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
				PrivateIPAddress: to.StringPtr("abc"),
				PublicIPAddress:  nil,
			},
		},
	}
	return &n.ApplicationGatewayPropertiesFormat{
		FrontendIPConfigurations: &feIPConfigs,
		Sku: &n.ApplicationGatewaySku{
			Name:     n.StandardV2,
			Tier:     n.ApplicationGatewayTierStandardV2,
			Capacity: to.Int32Ptr(3),
		},
	}
}

func newSecretStoreFixture(toAdd *map[string]interface{}) k8scontext.SecretsKeeper {
	c := cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{})
	ingressKey := getResourceKey(tests.Namespace, tests.Name)
	c.Add(ingressKey, tests.Host)

	key := tests.Namespace + "/" + tests.NameOfSecret
	c.Add(key, []byte("xyz"))

	if toAdd != nil {
		for k, v := range *toAdd {
			c.Add(k, v)
		}
	}

	return &k8scontext.SecretsStore{
		Cache: c,
	}
}

func keyFunc(obj interface{}) (string, error) {
	if service, ok := obj.(*v1.Service); ok {
		return fmt.Sprintf("%s/%s", service.Namespace, service.Name), nil
	}
	if endpoints, ok := obj.(*v1.Endpoints); ok {
		return fmt.Sprintf("%s/%s", endpoints.Namespace, endpoints.Name), nil
	}
	if ingress, ok := obj.(*v1beta1.Ingress); ok {
		return fmt.Sprintf("%s/%s", ingress.Namespace, ingress.Name), nil
	}
	return fmt.Sprintf("%s/%s", tests.Namespace, tests.ServiceName), nil
}

func newConfigBuilderFixture(certs *map[string]interface{}) appGwConfigBuilder {
	appGwConfig := NewAppGwyConfigFixture()
	cb := appGwConfigBuilder{
		appGwIdentifier: Identifier{
			SubscriptionID: tests.Subscription,
			ResourceGroup:  tests.ResourceGroup,
			AppGwName:      tests.AppGwName,
		},
		appGw: n.ApplicationGateway{ApplicationGatewayPropertiesFormat: appGwConfig},
		k8sContext: &k8scontext.Context{
			Caches: &k8scontext.CacheCollection{
				Endpoints: cache.NewStore(keyFunc),
				Secret:    cache.NewStore(keyFunc),
				Service:   cache.NewStore(keyFunc),
				Pods:      cache.NewStore(keyFunc),
				Ingress:   cache.NewStore(keyFunc),
			},
			CertificateSecretStore: newSecretStoreFixture(certs),
		},
		recorder: record.NewFakeRecorder(100),
	}

	return cb
}

func newCertsFixture() map[string]interface{} {
	toAdd := make(map[string]interface{})

	secretsIdent := secretIdentifier{
		Namespace: tests.Namespace,
		Name:      tests.Name,
	}

	toAdd[tests.Host] = secretsIdent
	toAdd[tests.OtherHost] = secretsIdent
	// Wild card
	toAdd[""] = secretsIdent

	return toAdd
}

func newURLPathMap() n.ApplicationGatewayURLPathMap {
	rule := n.ApplicationGatewayPathRule{
		ID:   to.StringPtr("-the-id-"),
		Type: to.StringPtr("-the-type-"),
		Etag: to.StringPtr("-the-etag-"),
		Name: to.StringPtr("/some/path"),
		ApplicationGatewayPathRulePropertiesFormat: &n.ApplicationGatewayPathRulePropertiesFormat{
			// A Path Rule must have either RedirectConfiguration xor (BackendAddressPool + BackendHTTPSettings)
			RedirectConfiguration: nil,

			BackendAddressPool:  resourceRef("--BackendAddressPool--"),
			BackendHTTPSettings: resourceRef("--BackendHTTPSettings--"),

			RewriteRuleSet:    resourceRef("--RewriteRuleSet--"),
			ProvisioningState: to.StringPtr("--provisionStateExpected--"),
		},
	}

	return n.ApplicationGatewayURLPathMap{
		Name: to.StringPtr("-path-map-name-"),
		ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{
			// URL Path Map must have either DefaultRedirectConfiguration xor (DefaultBackendAddressPool + DefaultBackendHTTPSettings)
			DefaultRedirectConfiguration: nil,

			DefaultBackendAddressPool:  resourceRef("--DefaultBackendAddressPool--"),
			DefaultBackendHTTPSettings: resourceRef("--DefaultBackendHTTPSettings--"),

			PathRules: &[]n.ApplicationGatewayPathRule{rule},
		},
	}
}
