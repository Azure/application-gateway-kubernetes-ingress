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
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"
)

const (
	testFixturesNamespace    = "--namespace--"
	testFixturesName         = "--name--"
	testFixturesHost         = "--some-hostname--"
	testFixturesOtherHost    = "--some-other-hostname--"
	testFixturesNameOfSecret = "--the-name-of-the-secret--"
	testFixturesServiceName  = "--service-name--"
	testFixturesNodeName     = "--node-name--"
	testFixturesURLPath      = "/a/b/c/d/e"
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
				Endpoints: cache.NewStore(func(obj interface{}) (string, error) {
					return "--namespace--/", nil
				}),
				Secret: cache.NewStore(func(obj interface{}) (string, error) {
					return "--namespace--/", nil
				}),
				Service: cache.NewStore(func(obj interface{}) (string, error) {
					return "--namespace--/", nil
				}),
				Pods: cache.NewStore(func(obj interface{}) (string, error) {
					return "--namespace--/", nil
				}),
			},
			CertificateSecretStore: makeSecretStoreTestFixture(certs),
		},
		probesMap: make(map[backendIdentifier](*network.ApplicationGatewayProbe)),
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
									Path: testFixturesURLPath,
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
									Path: testFixturesURLPath,
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
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.SslRedirectKey: "true",
			},
			Namespace: testFixturesNamespace,
			Name:      testFixturesName,
		},
	}
}

func makeServicePorts() (v1.ServicePort, v1.ServicePort, v1.ServicePort, v1.ServicePort) {
	port1 := v1.ServicePort{
		// The name of this port within the service. This must be a DNS_LABEL.
		// All ports within a ServiceSpec must have unique names. This maps to
		// the 'Name' field in EndpointPort objects.
		// Optional if only one ServicePort is defined on this service.
		Name: "http",

		// The IP protocol for this port. Supports "TCP", "UDP", and "SCTP".
		Protocol: v1.ProtocolTCP,

		// The port that will be exposed by this service.
		Port: 80,

		// Number or name of the port to access on the pods targeted by the service.
		// Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.
		// If this is a string, it will be looked up as a named port in the
		// target Pod's container ports. If this is not specified, the value
		// of the 'port' field is used (an identity map).
		// This field is ignored for services with clusterIP=None, and should be
		// omitted or set equal to the 'port' field.
		TargetPort: intstr.IntOrString{
			IntVal: 8181,
		},
	}

	port2 := v1.ServicePort{
		Name:     "https",
		Protocol: v1.ProtocolTCP,
		Port:     443,
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: "https-port",
		},
	}

	port3 := v1.ServicePort{
		Name:     "other-tcp-port",
		Protocol: v1.ProtocolTCP,
		Port:     554,
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 9554,
		},
	}

	port4 := v1.ServicePort{
		Name:     "other-tcp-port",
		Protocol: v1.ProtocolUDP,
		Port:     9123,
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 4566,
		},
	}

	return port1, port2, port3, port4
}

func makePod(serviceName string, ingressNamespace string, containerName string, containerPort int32) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ingressNamespace,
			Labels: map[string]string{
				"app": "frontend",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  serviceName,
					Image: "image",
					Ports: []v1.ContainerPort{
						{
							Name:          containerName,
							ContainerPort: containerPort,
						},
					},
					ReadinessProbe: &v1.Probe{
						TimeoutSeconds:   5,
						FailureThreshold: 3,
						PeriodSeconds:    20,
						Handler: v1.Handler{
							HTTPGet: &v1.HTTPGetAction{
								Host: "bye.com",
								Path: "/healthz",
								Port: intstr.IntOrString{
									Type:   intstr.String,
									StrVal: containerName,
								},
								Scheme: v1.URISchemeHTTP,
							},
						},
					},
				},
			},
		},
	}
}
