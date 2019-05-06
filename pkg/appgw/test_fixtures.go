// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"

	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func makeConfigBuilderTestFixture() appGwConfigBuilder {

	return appGwConfigBuilder{
		k8sContext: &k8scontext.Context{
			Caches: &k8scontext.CacheCollection{
				Secret: cache.NewStore(func(obj interface{}) (string, error) {
					return "", nil
				}),
			},
			CertificateSecretStore: nil,
		},
		secretIDCertificateMap: make(map[secretIdentifier]*string),
	}
}

func makeIngressTestFixture() v1beta1.Ingress {
	return v1beta1.Ingress{
		Spec: v1beta1.IngressSpec{
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
