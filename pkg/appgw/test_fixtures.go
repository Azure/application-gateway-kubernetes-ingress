// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"
)

const (
	testFixturesNamespace    = "--namespace--"
	testFixturesName         = "--name--"
	testFixturesHost         = "--some-hostname--"
	testFixturesOtherHost    = "--some-other-hostname--"
	testFixturesNameOfSecret = "--the-name-of-the-secret--"
)

func makeAppGwyConfigTestFixture() network.ApplicationGatewayPropertiesFormat {
	feIPConfigs := []network.ApplicationGatewayFrontendIPConfiguration{
		{
			Name: to.StringPtr("xx3"),
			Etag: to.StringPtr("xx2"),
			Type: to.StringPtr("xx1"),
			ID:   to.StringPtr("xx4"),
		},
		{
			Name: to.StringPtr("yy3"),
			Etag: to.StringPtr("yy2"),
			Type: to.StringPtr("yy1"),
			ID:   to.StringPtr("yy4"),
		},
	}
	return network.ApplicationGatewayPropertiesFormat{
		FrontendIPConfigurations: &feIPConfigs,
	}
}

func makeSecretStoreTestFixture(toAdd *map[string]interface{}) k8scontext.SecretsKeeper {
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

func makeConfigBuilderTestFixture(certs *map[string]interface{}) appGwConfigBuilder {
	cb := appGwConfigBuilder{
		appGwConfig:            makeAppGwyConfigTestFixture(),
		serviceBackendPairMap:  make(map[backendIdentifier]serviceBackendPortPair),
		backendHTTPSettingsMap: make(map[backendIdentifier]*network.ApplicationGatewayBackendHTTPSettings),
		backendPoolMap:         make(map[backendIdentifier]*network.ApplicationGatewayBackendAddressPool),
		k8sContext: &k8scontext.Context{
			Caches: &k8scontext.CacheCollection{
				Secret: cache.NewStore(func(obj interface{}) (string, error) {
					return "", nil
				}),
				Service: cache.NewStore(func(obj interface{}) (string, error) {
					return "", nil
				}),
			},
			CertificateSecretStore: makeSecretStoreTestFixture(certs),
		},
		probesMap:                     make(map[backendIdentifier](*network.ApplicationGatewayProbe)),
	}

	return cb
}

func getCertsTestFixture() map[string]interface{} {
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

func makeIngressTestFixture() v1beta1.Ingress {
	return v1beta1.Ingress{
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: testFixturesHost,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/a/b/c/d/e",
									Backend: v1beta1.IngressBackend{
										ServiceName: "",
										ServicePort: intstr.IntOrString{
											IntVal: 8080,
										},
									},
								},
							},
						},
					},
				},
				{
					Host: testFixturesOtherHost,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/a/b/c/d/e",
									Backend: v1beta1.IngressBackend{
										ServiceName: "",
										ServicePort: intstr.IntOrString{
											IntVal: 8989,
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []v1beta1.IngressTLS{
				{
					Hosts: []string{
						"www.contoso.com",
						"ftp.contoso.com",
						testFixturesHost,
						"",
					},
					SecretName: testFixturesNameOfSecret,
				},
				{
					Hosts:      []string{},
					SecretName: testFixturesNameOfSecret,
				},
			},
		},
		ObjectMeta: v1.ObjectMeta{
			Annotations: map[string]string{
				annotations.SslRedirectKey: "true",
			},
			Namespace: testFixturesNamespace,
			Name:      testFixturesName,
		},
	}
}
