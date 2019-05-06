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
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func makeConfigBuilderTestFixture() appGwConfigBuilder {

	cb := appGwConfigBuilder{
		serviceBackendPairMap:         make(map[backendIdentifier]serviceBackendPortPair),
		httpListenersMap:              make(map[frontendListenerIdentifier]*network.ApplicationGatewayHTTPListener),
		httpListenersAzureConfigMap:   make(map[frontendListenerIdentifier]*frontendListenerAzureConfig),
		ingressKeyHostnameSecretIDMap: make(map[string]map[string]secretIdentifier),
		secretIDCertificateMap:        make(map[secretIdentifier]*string),
		backendHTTPSettingsMap:        make(map[backendIdentifier]*network.ApplicationGatewayBackendHTTPSettings),
		backendPoolMap:                make(map[backendIdentifier]*network.ApplicationGatewayBackendAddressPool),
		k8sContext: &k8scontext.Context{
			Caches: &k8scontext.CacheCollection{
				Secret: cache.NewStore(func(obj interface{}) (string, error) {
					return "", nil
				}),
			},
			CertificateSecretStore: nil,
		},
	}

	return cb
}

func addCertsTestFixture(cb *appGwConfigBuilder) {
	ingressKey := "--ingress-key--"
	hostName := "--some-hostname--"
	secretsIdent := secretIdentifier{
		Namespace: "--namespace--",
		Name:      "--name--",
	}
	cb.ingressKeyHostnameSecretIDMap[ingressKey] = make(map[string]secretIdentifier)
	cb.ingressKeyHostnameSecretIDMap[ingressKey][hostName] = secretsIdent
	// Wild card
	cb.ingressKeyHostnameSecretIDMap[ingressKey][""] = secretsIdent

	cb.secretIDCertificateMap[secretsIdent] = to.StringPtr("")
}

func makeIngressTestFixture() v1beta1.Ingress {
	return v1beta1.Ingress{
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "-some-host-",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "////",
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
			},
			TLS: []v1beta1.IngressTLS{
				{
					Hosts: []string{
						"www.contoso.com",
						"ftp.contoso.com",
					},
					SecretName: "--the-name-of-the-secret--",
				},
			},
		},
		ObjectMeta: v1.ObjectMeta{
			Annotations: map[string]string{
				annotations.SslRedirectKey: "true",
			},
			Namespace: "--ingress--namespace--",
			Name:      "--ingress-name--",
		},
	}
}
