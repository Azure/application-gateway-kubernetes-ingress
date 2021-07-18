// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package tests

import (
	"fmt"
	"io/ioutil"

	"github.com/Azure/go-autorest/autorest/to"
	n "github.com/akshaysngupta/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"

	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
)

// constant values to be used for testing
const (
	Namespace                    = "--namespace--"
	OtherNamespace               = "--other-namespace--"
	HTTPSBackendNamespace        = "--https-backend-namespace--"
	Name                         = "--name--"
	Host                         = "bye.com"
	OtherHost                    = "--some-other-hostname--"
	HostUnassociated             = "---some-host-without-routing-rules---"
	WildcardHost1                = "*.hi.com"
	WildcardHost2                = "hola.com"
	NameOfSecret                 = "--the-name-of-the-secret--"
	ServiceName                  = "--service-name--"
	LoadDistributionPolicyName   = "--load-dist-policy--"
	NodeName                     = "--node-name--"
	URLPath1                     = "/api1"
	URLPath2                     = "/api2"
	URLPath3                     = "/api3"
	URLPath4                     = "/"
	URLPath5                     = "/*"
	HealthPath                   = "/healthz"
	ContainerName                = "--container-name--"
	ContainerPort                = int32(9876)
	ContainerHealthPortName      = "--container-health-port-name--"
	ContainerHealthHTTPSPortName = "--container-health-https-port-name--"
	ContainerHealthHTTPSPort     = int32(443)
	ContainerHealthPort          = int32(9090)
	ServicePort                  = "service-port"
	SelectorKey                  = "app"
	SelectorValue                = "frontend"
	Subscription                 = "--subscription--"
	ResourceGroup                = "--resource-group--"
	AppGwName                    = "--app-gw-name--"
	PublicIPID                   = "--front-end-ip-id-1--"
	PrivateIPID                  = "--front-end-ip-id-2--"
	ServiceHTTPPort              = "--service-http-port--"
	ServiceHTTPSPort             = "--service-https-port--"
)

// GetIngress creates an Ingress test fixture.
func GetIngress() (*networking.Ingress, error) {
	return GetIngressV1FromFile("ingress.yaml")
}

// GetVerySimpleIngress creates one very simple Ingress test fixture with no rules.
func GetVerySimpleIngress() *networking.Ingress {
	return &networking.Ingress{
		Spec: networking.IngressSpec{
			DefaultBackend: &networking.IngressBackend{
				Service: &networking.IngressServiceBackend{
					Name: "websocket-service",
					Port: networking.ServiceBackendPort{
						Number: 80,
					},
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
			Name: "websocket-ingress",
		},
	}
}

// GetIngressComplex creates an Ingress test fixture with multiple backends and path rules.
func GetIngressComplex() (*networking.Ingress, error) {
	return GetIngressV1FromFile("ingress-complex.yaml")
}

// GetIngressWithMissingServiceAndServiceWithInvalidPort with missing service and service with invalid port
func GetIngressWithMissingServiceAndServiceWithInvalidPort() *networking.Ingress {
	be70000 := NewIngressBackendFixture("--service-name--", 70000)
	return &networking.Ingress{
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				NewIngressRuleFixture("--service-name--", "/otherService/*", *be70000),
			},
			DefaultBackend: &networking.IngressBackend{
				Service: &networking.IngressServiceBackend{
					Name: "missing-service",
					Port: networking.ServiceBackendPort{
						Number: 8080,
					},
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
			Namespace: Namespace,
			Name:      "ingress-with-invalid-services",
		},
	}
}

// GetIngressNamespaced creates 2 Ingress test fixtures in different namespaces.
func GetIngressNamespaced() (*[]networking.Ingress, error) {
	ingr1, err := GetIngressV1FromFile("ingress-namespace-1.yaml")
	if err != nil {
		klog.Fatal(err)
	}
	ingr2, err := GetIngressV1FromFile("ingress-namespace-2.yaml")
	if err != nil {
		klog.Fatal(err)
	}
	return &[]networking.Ingress{*ingr1, *ingr2}, nil
}

// GetIngressV1FromFile reads an ingress V1 from file
func GetIngressV1FromFile(fileName string) (*networking.Ingress, error) {
	ingr, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Print(err)
	}

	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(ingr, nil, nil)
	if err != nil {
		return nil, err
	}
	return obj.(*networking.Ingress), nil
}

// GetIngressV1Beta1FromFile reads an ingress V1Beta1 from file
func GetIngressV1Beta1FromFile(fileName string) (*extensions.Ingress, error) {
	ingr, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Print(err)
	}

	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(ingr, nil, nil)
	if err != nil {
		return nil, err
	}
	return obj.(*extensions.Ingress), nil
}

// GetApplicationGatewayBackendAddressPool makes a new ApplicationGatewayBackendAddressPool for testing.
func GetApplicationGatewayBackendAddressPool() *n.ApplicationGatewayBackendAddressPool {
	return &n.ApplicationGatewayBackendAddressPool{
		Name: to.StringPtr("defaultaddresspool"),
		Etag: nil,
		Type: nil,
		ID:   nil,
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendIPConfigurations: nil,
			BackendAddresses:        &[]n.ApplicationGatewayBackendAddress{},
			ProvisioningState:       "",
		},
	}
}

// NewIngressBackendFixture makes a new Ingress Backend for testing
func NewIngressBackendFixture(serviceName string, port int32) *networking.IngressBackend {
	return &networking.IngressBackend{
		Service: &networking.IngressServiceBackend{
			Name: serviceName,
			Port: networking.ServiceBackendPort{
				Number: port,
			},
		},
	}
}

func NewIngressLoadDistributionPolicyBackendFixture(ldpName string) *networking.IngressBackend {
	return &networking.IngressBackend{
		Resource: &v1.TypedLocalObjectReference{
			APIGroup: to.StringPtr("appgw.ingress.azure.io"),
			Kind:     "AzureApplicationGatewayLoadDistributionPolicy",
			Name:     ldpName,
		},
	}
}

// NewIngressRuleFixture makes a new Ingress Rule for testing
func NewIngressRuleFixture(host string, urlPath string, be networking.IngressBackend) networking.IngressRule {
	return networking.IngressRule{
		Host: host,
		IngressRuleValue: networking.IngressRuleValue{
			HTTP: &networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{
					{
						Path:    urlPath,
						Backend: be,
					},
				},
			},
		},
	}
}

// NewIngressRuleWithPathsFixture makes a new Ingress Rule with mutiple paths for testing
func NewIngressRuleWithPathsFixture(host string, urlPathsList []string, be networking.IngressBackend) networking.IngressRule {
	ingressRule := networking.IngressRule{
		Host: host,
		IngressRuleValue: networking.IngressRuleValue{
			HTTP: &networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{},
			},
		},
	}

	for _, value := range urlPathsList {
		ingressRule.IngressRuleValue.HTTP.Paths = append(ingressRule.IngressRuleValue.HTTP.Paths, networking.HTTPIngressPath{
			Path:    value,
			Backend: be,
		})
	}

	return ingressRule
}

func NewIngressWithLoadDistributionPolicyFixture() *networking.Ingress {
	ldp443 := NewIngressLoadDistributionPolicyBackendFixture(LoadDistributionPolicyName)
	return &networking.Ingress{
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				NewIngressRuleFixture(Host, URLPath2, *ldp443),
			},
			TLS: []networking.IngressTLS{
				{
					Hosts: []string{
						"www.contoso.com",
						"ftp.contoso.com",
						Host,
						"",
					},
					SecretName: NameOfSecret,
				},
				{
					Hosts:      []string{},
					SecretName: NameOfSecret,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				annotations.SslRedirectKey:  "true",
			},
			Namespace: Namespace,
			Name:      Name,
		},
	}
}

// NewIngressFixture makes a new Ingress for testing
func NewIngressFixture() *networking.Ingress {
	be80 := NewIngressBackendFixture(ServiceName, 80)
	be443 := NewIngressBackendFixture(ServiceName, 443)

	return &networking.Ingress{
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				NewIngressRuleFixture(Host, URLPath1, *be80),
				NewIngressRuleFixture(Host, URLPath2, *be443),
			},
			TLS: []networking.IngressTLS{
				{
					Hosts: []string{
						"www.contoso.com",
						"ftp.contoso.com",
						Host,
						"",
					},
					SecretName: NameOfSecret,
				},
				{
					Hosts:      []string{},
					SecretName: NameOfSecret,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				annotations.SslRedirectKey:  "true",
			},
			Namespace: Namespace,
			Name:      Name,
		},
	}
}

// NewIngressFixtureSingleSlashPath makes a new Ingress with "/" and "/*" as ingress rule path for testing
func NewIngressFixtureSingleSlashPath() *networking.Ingress {
	be80 := NewIngressBackendFixture(ServiceName, 80)
	be443 := NewIngressBackendFixture(ServiceName, 443)

	return &networking.Ingress{
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				NewIngressRuleFixture(Host, URLPath4, *be80),
				NewIngressRuleFixture(Host, URLPath5, *be443),
			},
			TLS: []networking.IngressTLS{
				{
					Hosts: []string{
						"www.contoso.com",
						"ftp.contoso.com",
						Host,
						"",
					},
					SecretName: NameOfSecret,
				},
				{
					Hosts:      []string{},
					SecretName: NameOfSecret,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
				annotations.SslRedirectKey:  "true",
			},
			Namespace: Namespace,
			Name:      Name,
		},
	}
}

// NewServicePortsFixture makes a new service port for testing
func NewServicePortsFixture() *[]v1.ServicePort {
	httpPort := v1.ServicePort{
		// The name of this port within the service. This must be a DNS_LABEL.
		// All ports within a ServiceSpec must have unique names. This maps to
		// the 'Name' field in EndpointPort objects.
		// Optional if only one ServicePort is defined on this service.
		Name: ServiceHTTPPort,

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
			Type:   intstr.Int,
			IntVal: ContainerPort,
		},
	}

	httpsPort := v1.ServicePort{
		Name:     ServiceHTTPSPort,
		Protocol: v1.ProtocolTCP,
		Port:     443,
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: "https-port",
		},
	}

	randomTCPPort := v1.ServicePort{
		Name:     "other-tcp-port",
		Protocol: v1.ProtocolTCP,
		Port:     554,
		TargetPort: intstr.IntOrString{
			IntVal: 8181,
		},
	}

	udpPort := v1.ServicePort{
		Name:     "other-tcp-port",
		Protocol: v1.ProtocolUDP,
		Port:     9123,
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: 4566,
		},
	}

	return &[]v1.ServicePort{
		httpPort,
		httpsPort,
		randomTCPPort,
		udpPort,
	}
}

// NewProbeFixture makes a new probe for testing
func NewProbeFixture(containerName string) *v1.Probe {
	return &v1.Probe{
		TimeoutSeconds:   5,
		FailureThreshold: 3,
		PeriodSeconds:    20,
		Handler: v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Host: Host,
				Path: HealthPath,
				Port: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: ContainerHealthPortName,
				},
				Scheme: v1.URISchemeHTTP,
			},
		},
	}
}

// NewHTTPSProbeFixture makes a new probe for testing port 443
func NewHTTPSProbeFixture(containerName string) *v1.Probe {
	return &v1.Probe{
		TimeoutSeconds:   5,
		FailureThreshold: 3,
		PeriodSeconds:    20,
		Handler: v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path: HealthPath,
				Port: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: ContainerHealthHTTPSPortName,
				},
			},
		},
	}
}

// NewPodHTTPSFixture makes a new pod serving at port 443
func NewPodHTTPSFixture(serviceName string, ingressNamespace string, containerName string, containerPort int32) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ingressNamespace,
			Labels: map[string]string{
				SelectorKey: SelectorValue,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  serviceName,
					Image: "image",
					Ports: []v1.ContainerPort{
						{
							Name:          ContainerHealthHTTPSPortName,
							ContainerPort: ContainerHealthHTTPSPort,
						},
					},
					ReadinessProbe: NewHTTPSProbeFixture(containerName),
					LivenessProbe:  NewHTTPSProbeFixture(containerName),
				},
			},
		},
	}
}

// NewPodFixture makes a new pod for testing
func NewPodFixture(serviceName string, ingressNamespace string, containerName string, containerPort int32) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: ingressNamespace,
			Labels: map[string]string{
				SelectorKey: SelectorValue,
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
						{
							Name:          ContainerHealthPortName,
							ContainerPort: ContainerHealthPort,
						},
					},
					ReadinessProbe: NewProbeFixture(containerName),
					LivenessProbe:  NewProbeFixture(containerName),
				},
			},
		},
	}
}

// NewServiceFixture makes a new service for testing
func NewServiceFixture(servicePorts ...v1.ServicePort) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: Namespace,
		},
		Spec: v1.ServiceSpec{
			// The list of ports that are exposed by this service.
			Ports: servicePorts,

			// Route service traffic to pods with label keys and values matching this
			// selector. If empty or not present, the service is assumed to have an
			// external process managing its endpoints, which Kubernetes will not
			// modify. Only applies to types ClusterIP, NodePort, and LoadBalancer.
			// Ignored if type is ExternalName.
			Selector: map[string]string{
				SelectorKey: SelectorValue,
			},
		},
	}
}

// NewEndpointsFixtureWithSameNameMultiplePorts makes a new endpoint for testing
func NewEndpointsFixtureWithSameNameMultiplePorts() *v1.Endpoints {
	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: Namespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				// IP addresses which offer the related ports that are marked as ready. These endpoints
				// should be considered safe for load balancers and clients to utilize.
				// +optional
				Addresses: []v1.EndpointAddress{
					{
						IP: "10.9.8.7",
						// The Hostname of this endpoint
						// +optional
						Hostname: "www.contoso.com",
						// Optional: Node hosting this endpoint. This can be used to determine endpoints local to a node.
						// +optional
						NodeName: to.StringPtr(NodeName),
					},
				},
				// IP addresses which offer the related ports but are not currently marked as ready
				// because they have not yet finished starting, have recently failed a readiness check,
				// or have recently failed a liveness check.
				// +optional
				NotReadyAddresses: []v1.EndpointAddress{},
				// Port numbers available on the related IP addresses.
				// +optional
				Ports: []v1.EndpointPort{
					{
						Protocol: v1.ProtocolTCP,
						Name:     ServiceHTTPSPort,
						Port:     5000,
					},
					{
						Protocol: v1.ProtocolTCP,
						Name:     ServiceHTTPSPort,
						Port:     80,
					},
					{
						Protocol: v1.ProtocolTCP,
						Name:     ServiceHTTPSPort,
						Port:     81,
					},
					{
						Protocol: v1.ProtocolTCP,
						Name:     ServiceHTTPSPort,
						Port:     4999,
					},
					{
						Protocol: v1.ProtocolTCP,
						Name:     ServiceHTTPSPort,
						Port:     75,
					},
				},
			},
		},
	}
}

// NewEndpointsFixture makes a new endpoint for testing
func NewEndpointsFixture() *v1.Endpoints {
	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: Namespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				// IP addresses which offer the related ports that are marked as ready. These endpoints
				// should be considered safe for load balancers and clients to utilize.
				// +optional
				Addresses: []v1.EndpointAddress{
					{
						IP: "10.9.8.7",
						// The Hostname of this endpoint
						// +optional
						Hostname: "www.contoso.com",
						// Optional: Node hosting this endpoint. This can be used to determine endpoints local to a node.
						// +optional
						NodeName: to.StringPtr(NodeName),
					},
				},
				// IP addresses which offer the related ports but are not currently marked as ready
				// because they have not yet finished starting, have recently failed a readiness check,
				// or have recently failed a liveness check.
				// +optional
				NotReadyAddresses: []v1.EndpointAddress{},
				// Port numbers available on the related IP addresses.
				// +optional
				Ports: []v1.EndpointPort{
					{
						Protocol: v1.ProtocolTCP,
						Name:     ServiceHTTPPort,
						Port:     ContainerPort,
					},
					{
						Protocol: v1.ProtocolTCP,
						Name:     ServiceHTTPSPort,
						Port:     ContainerPort,
					},
				},
			},
		},
	}
}

// NewIngressTestFixture creates a new Ingress struct for testing.
func NewIngressTestFixture(namespace string, ingressName string) networking.Ingress {
	return networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "hello.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: "/hi",
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: ServiceName,
											Port: networking.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// NewIngressTestFixtureBasic creates a basic Ingress with / path for testing.
func NewIngressTestFixtureBasic(namespace string, ingressName string, tls bool) *networking.Ingress {
	ingress := &networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "hello.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: "/",
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: ServiceName,
											Port: networking.ServiceBackendPort{
												Name: "http",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if tls {
		ingress.Spec.TLS = []networking.IngressTLS{
			{
				SecretName: NameOfSecret,
			},
		}
		ingress.Annotations[annotations.SslRedirectKey] = "true"
	}

	return ingress
}

// NewPodTestFixture creates a new Pod struct for testing.
func NewPodTestFixture(namespace string, podName string) v1.Pod {
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				SelectorKey: SelectorValue,
			},
		},
		Spec: v1.PodSpec{},
	}
}

// NewSecretTestFixture creates a new secret for testing
func NewSecretTestFixture() *v1.Secret {
	secret := &v1.Secret{
		Type: "kubernetes.io/tls",
		ObjectMeta: metav1.ObjectMeta{
			Name:      NameOfSecret,
			Namespace: Namespace,
		},
		Data: make(map[string][]byte),
	}

	tlsKey := "tls.key"
	tlsCrt := "tls.crt"

	secret.Data[tlsKey] = []byte("-----BEGIN PRIVATE KEY-----\n" +
		"MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDFEs4opOIMHYna\n" +
		"wMio1JHZaQWDZEP8fsL23Rozhow0vVokthPk4wGBKYpc8XYBbWFs5pUuExeOjeRW\n" +
		"jdNArwn5jCZYaxtqfdrj2kLHFHPCTwmbzn+qvPkvp/ZJyeY+4eIe7soGzO6hoj/w\n" +
		"HHdry7rPiap5R5EMfHzfl1TZ5WfixqnxKVEc33VRD9xwQIHwJTGnoI2bTGK3vK5q\n" +
		"90Glxyc4FAqo6xBguo6ZfOCqPYHXAKtaMj5hcr2dA0/7rJ/xNthDdnQwhETU2BgQ\n" +
		"9PvqfuMif+r/VM4KmYjQYu6+NN8VDVq6eSx4dxIzqWZ/NdSeoIri+6Gpa0AncMrq\n" +
		"3t7OjuQjAgMBAAECggEAKfzOtbQjgSdN9rB6UBYyGNsaVJspLQOo8EW9TlsNRjNN\n" +
		"oGK2rF59NJKwKws69CTky/n4sL9aloG+s342EyL4AhYNGWuAhNjZqRAYiCfgXfbO\n" +
		"+kYtxyfKA5BKlgARMTaZIbQIkRhag095ReQawXm/jHYtPvezfLCNPmoUpvQMhTEk\n" +
		"jzhhB7Ao5JPkw6jjnYa4raETYR3LTdFwhfU1WecEJ+Mj1hGX8ANC8cdHYxvkomcl\n" +
		"/ucl99siNJKYHZ6wWXpLRICZyTyLCcCnICj2g/+8BiV9pokrUHYW5diLDN4UBHnQ\n" +
		"Qe2LZnC+hIU8Vvq2z9Wy8tF8Z2LMmswK+kIff7tNuQKBgQDxil1AaMSCAbTecErf\n" +
		"RJkK81YvtMvM5ha2lhHOdnGvl/aVMdQ1rAkklGXMbIz/e87gOR3PfbmY67QR9aEz\n" +
		"CTXjfWG6J5Ri99kEn3af3AOrbJ6dbgaZWKwvtuDfXciuFo+0K5eQcC6r5Cr5Wjs3\n" +
		"DAnYWMGz9sUyMg0s6OWqifLMjQKBgQDQ3vqOXVzjRbvhyq/QWerl/x3jUIs/fY3e\n" +
		"6IAcf7jyihvQWAX361yjQig8n8D6XcPo1GKmKr93ra7cVdH3eQ1ICG9KzfMyG0PQ\n" +
		"H6qkft19BAwsCK687LeotTGX4qXavXG9AP8tLyq8WRGdmQPStpses7oUjOBk3rKo\n" +
		"8puKExe/bwKBgQDAupvf2fj6l2v/lXBYqH7JexLJLCT2EJ4NAL+ik2XxK3tI3qKq\n" +
		"VORSuMpljDQRY3PV/B0qQ/KE74YWUn1WoMHMDG6fQBepxIP4qVjZA5A2B4ykp3dC\n" +
		"gruZsv3JnSaUqlHt/F6KlMjYxU34+yOGr+dnJqMg+wWsIL3cmNUw97OxvQKBgCo4\n" +
		"6O1ecih/MDu0fVXg11sm9yO8ZGmxN7yXw03/g6ODx5uWL56uNUvLU9btdFUoHzIx\n" +
		"vL9aZNoMggyITKl6DvVAvz6f40l9uXeY7yXRf3SGHO/J0YjfUUEJX70UU/Kj2Rob\n" +
		"2XmIz1rDpov1IpC12SWbr0H4OGQroHIGmOqQcXyBAoGAMFJzs9K/bx6MA9lZW8Vw\n" +
		"adbuUcqFjOAryk8fNzZNBgYADRHfNz1Az3vqIi1zWcnimou0M4o2BRGUc805wy7V\n" +
		"YfkIyRQ5bIIVGNpP19dEOSsJ8pYAr+Bo/3GjXxUe6O6PxF3hbfPJNWt11refYC27\n" +
		"dZsRsRJX4pAw+BznAZodf6Q=\n" +
		"-----END PRIVATE KEY-----\n")
	secret.Data[tlsCrt] = []byte("-----BEGIN CERTIFICATE-----\n" +
		"MIIDazCCAlOgAwIBAgIUOX75BZ3gP92zRT89ZO34HXdi44QwDQYJKoZIhvcNAQEL\n" +
		"BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM\n" +
		"GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0xOTA4MjkwMDQ0NDdaFw0yMDA4\n" +
		"MjgwMDQ0NDdaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw\n" +
		"HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB\n" +
		"AQUAA4IBDwAwggEKAoIBAQDFEs4opOIMHYnawMio1JHZaQWDZEP8fsL23Rozhow0\n" +
		"vVokthPk4wGBKYpc8XYBbWFs5pUuExeOjeRWjdNArwn5jCZYaxtqfdrj2kLHFHPC\n" +
		"Twmbzn+qvPkvp/ZJyeY+4eIe7soGzO6hoj/wHHdry7rPiap5R5EMfHzfl1TZ5Wfi\n" +
		"xqnxKVEc33VRD9xwQIHwJTGnoI2bTGK3vK5q90Glxyc4FAqo6xBguo6ZfOCqPYHX\n" +
		"AKtaMj5hcr2dA0/7rJ/xNthDdnQwhETU2BgQ9PvqfuMif+r/VM4KmYjQYu6+NN8V\n" +
		"DVq6eSx4dxIzqWZ/NdSeoIri+6Gpa0AncMrq3t7OjuQjAgMBAAGjUzBRMB0GA1Ud\n" +
		"DgQWBBTCTeqqryPyXKMAoo28CGKvS2dvuDAfBgNVHSMEGDAWgBTCTeqqryPyXKMA\n" +
		"oo28CGKvS2dvuDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQCp\n" +
		"e7uP6D0bU6Z/ZuWZrZUwvo054912wg7O7zNJeZ1dnV9M/3ozR5UR1LSilhRgtOLD\n" +
		"mUIQtQdoJCTnPb/FrD7ZvOL5e0CjbvKSs7UxhvsOBiE4EQCHS4Gp1HUtFRS+H60U\n" +
		"Z0cUG4CnbjJy0JmXpEq+B1McDc7QtR9p0JJiOIJN59255u/Kdg+0NWdRsB6zdZMn\n" +
		"p4gifcw3N8eErYFSs6mHhblTOROMf0kCGan6qyx08Lk/t3YI33ZAktk8T5GVSe3A\n" +
		"o1nu88fKxKLEH6kcBzx35dt3CmMsHCXgX58R+OHD8boJteLkkuc+h+mzO7G8h/Bv\n" +
		"LloWsUALcQTN0LMl33F8\n" +
		"-----END CERTIFICATE-----\n")

	return secret
}
