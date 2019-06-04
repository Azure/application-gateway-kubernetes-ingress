// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/client-go/tools/cache"
)

const (
	testFixturesNamespace     = "--namespace--"
	testFixturesName          = "--name--"
	testFixturesHost          = "bye.com"
	testFixturesOtherHost     = "--some-other-hostname--"
	testFixturesNameOfSecret  = "--the-name-of-the-secret--"
	testFixturesServiceName   = "--service-name--"
	testFixturesNodeName      = "--node-name--"
	testFixturesURLPath       = "/healthz"
	testFixturesContainerName = "--container-name--"
	testFixturesContainerPort = int32(9876)
	testFixturesServicePort   = "service-port"
	testFixturesSelectorKey   = "app"
	testFixturesSelectorValue = "frontend"
	testFixtureSubscription   = "--subscription--"
	testFixtureResourceGroup  = "--resource-group--"
	testFixtureAppGwName      = "--app-gw-name--"
	testFixtureIPID1          = "--front-end-ip-id-1--"
	testFixturesSubscription  = "--subscription--"
)

func NewAppGwyConfigFixture() network.ApplicationGatewayPropertiesFormat {
	feIPConfigs := []network.ApplicationGatewayFrontendIPConfiguration{
		{
			// Private IP
			Name: to.StringPtr("xx3"),
			Etag: to.StringPtr("xx2"),
			Type: to.StringPtr("xx1"),
			ID:   to.StringPtr(testFixtureIPID1),
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

func NewSecretStoreFixture(toAdd *map[string]interface{}) k8scontext.SecretsKeeper {
	c := cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{})
	ingressKey := getResourceKey(testFixturesNamespace, testFixturesName)
	c.Add(ingressKey, testFixturesHost)

	key := testFixturesNamespace + "/" + testFixturesNameOfSecret
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

func KeyFunc(obj interface{}) (string, error) {
	return fmt.Sprintf("%s/%s", testFixturesNamespace, testFixturesServiceName), nil
}

func NewConfigBuilderFixture(certs *map[string]interface{}) appGwConfigBuilder {
	cb := appGwConfigBuilder{
		appGwIdentifier: Identifier{
			SubscriptionID: testFixtureSubscription,
			ResourceGroup:  testFixtureResourceGroup,
			AppGwName:      testFixtureAppGwName,
		},
		appGwConfig:            NewAppGwyConfigFixture(),
		serviceBackendPairMap:  make(map[backendIdentifier]serviceBackendPortPair),
		backendHTTPSettingsMap: make(map[backendIdentifier]*network.ApplicationGatewayBackendHTTPSettings),
		backendPoolMap:         make(map[backendIdentifier]*network.ApplicationGatewayBackendAddressPool),
		k8sContext: &k8scontext.Context{
			Caches: &k8scontext.CacheCollection{
				Endpoints: cache.NewStore(KeyFunc),
				Secret:    cache.NewStore(KeyFunc),
				Service:   cache.NewStore(KeyFunc),
				Pods:      cache.NewStore(KeyFunc),
			},
			CertificateSecretStore: NewSecretStoreFixture(certs),
		},
		probesMap: make(map[backendIdentifier]*network.ApplicationGatewayProbe),
		recorder:  record.NewFakeRecorder(1),
	}

	return cb
}

func NewCertsFixture() map[string]interface{} {
	toAdd := make(map[string]interface{})

	secretsIdent := secretIdentifier{
		Namespace: testFixturesNamespace,
		Name:      testFixturesName,
	}

	toAdd[testFixturesHost] = secretsIdent
	toAdd[testFixturesOtherHost] = secretsIdent
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
