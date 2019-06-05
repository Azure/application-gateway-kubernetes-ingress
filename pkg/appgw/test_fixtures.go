// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/client-go/tools/cache"
)

func newAppGwyConfigFixture() network.ApplicationGatewayPropertiesFormat {
	feIPConfigs := []network.ApplicationGatewayFrontendIPConfiguration{
		{
			// Private IP
			Name: to.StringPtr("xx3"),
			Etag: to.StringPtr("xx2"),
			Type: to.StringPtr("xx1"),
			ID:   to.StringPtr(tests.IPID1),
			ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &network.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
				PrivateIPAddress: nil,
				PublicIPAddress: &network.SubResource{
					ID: to.StringPtr("xyz"),
				},
			},
		},
		{
			// Public IP
			Name: to.StringPtr("yy3"),
			Etag: to.StringPtr("yy2"),
			Type: to.StringPtr("yy1"),
			ID:   to.StringPtr("yy4"),
			ApplicationGatewayFrontendIPConfigurationPropertiesFormat: &network.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
				PrivateIPAddress: to.StringPtr("abc"),
				PublicIPAddress:  nil,
			},
		},
	}
	return network.ApplicationGatewayPropertiesFormat{
		FrontendIPConfigurations: &feIPConfigs,
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
	return fmt.Sprintf("%s/%s", tests.Namespace, tests.ServiceName), nil
}

func newConfigBuilderFixture(certs *map[string]interface{}) appGwConfigBuilder {
	cb := appGwConfigBuilder{
		appGwIdentifier: Identifier{
			SubscriptionID: tests.Subscription,
			ResourceGroup:  tests.ResourceGroup,
			AppGwName:      tests.AppGwName,
		},
		appGwConfig:            newAppGwyConfigFixture(),
		serviceBackendPairMap:  make(map[backendIdentifier]serviceBackendPortPair),
		backendHTTPSettingsMap: make(map[backendIdentifier]*network.ApplicationGatewayBackendHTTPSettings),
		k8sContext: &k8scontext.Context{
			Caches: &k8scontext.CacheCollection{
				Endpoints: cache.NewStore(keyFunc),
				Secret:    cache.NewStore(keyFunc),
				Service:   cache.NewStore(keyFunc),
				Pods:      cache.NewStore(keyFunc),
			},
			CertificateSecretStore: newSecretStoreFixture(certs),
		},
		probesMap: make(map[backendIdentifier]*network.ApplicationGatewayProbe),
		recorder:  record.NewFakeRecorder(1),
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

func newURLPathMap() network.ApplicationGatewayURLPathMap {
	rule := network.ApplicationGatewayPathRule{
		ID:   to.StringPtr("-the-id-"),
		Type: to.StringPtr("-the-type-"),
		Etag: to.StringPtr("-the-etag-"),
		Name: to.StringPtr("/some/path"),
		ApplicationGatewayPathRulePropertiesFormat: &network.ApplicationGatewayPathRulePropertiesFormat{
			// A Path Rule must have either RedirectConfiguration xor (BackendAddressPool + BackendHTTPSettings)
			RedirectConfiguration: nil,

			BackendAddressPool:  resourceRef("--BackendAddressPool--"),
			BackendHTTPSettings: resourceRef("--BackendHTTPSettings--"),

			RewriteRuleSet:    resourceRef("--RewriteRuleSet--"),
			ProvisioningState: to.StringPtr("--provisionStateExpected--"),
		},
	}

	return network.ApplicationGatewayURLPathMap{
		Name: to.StringPtr("-path-map-name-"),
		ApplicationGatewayURLPathMapPropertiesFormat: &network.ApplicationGatewayURLPathMapPropertiesFormat{
			// URL Path Map must have either DefaultRedirectConfiguration xor (DefaultBackendAddressPool + DefaultBackendHTTPSettings)
			DefaultRedirectConfiguration: nil,

			DefaultBackendAddressPool:  resourceRef("--DefaultBackendAddressPool--"),
			DefaultBackendHTTPSettings: resourceRef("--DefaultBackendHTTPSettings--"),

			PathRules: &[]network.ApplicationGatewayPathRule{rule},
		},
	}
}
