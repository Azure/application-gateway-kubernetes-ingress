// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package tests

import (
	"fmt"

	"io/ioutil"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
)

// constant values to be used for testing
const (
	Namespace        = "--namespace--"
	Name             = "--name--"
	Host             = "bye.com"
	OtherHost        = "--some-other-hostname--"
	HostUnassociated = "---some-host-without-routing-rules---"
	NameOfSecret     = "--the-name-of-the-secret--"
	ServiceName      = "--service-name--"
	NodeName         = "--node-name--"
	URLPath1         = "/api1"
	URLPath2         = "/api2"
	URLPath3         = "/api3"
	HealthPath       = "/healthz"
	ContainerName    = "--container-name--"
	ContainerPort    = int32(9876)
	ServicePort      = "service-port"
	SelectorKey      = "app"
	SelectorValue    = "frontend"
	Subscription     = "--subscription--"
	ResourceGroup    = "--resource-group--"
	AppGwName        = "--app-gw-name--"
	PublicIPID       = "--front-end-ip-id-1--"
	PrivateIPID      = "--front-end-ip-id-2--"
	ServiceHTTPPort  = "--service-http-port--"
	ServiceHTTPSPort = "--service-https-port--"
)

// GetIngress creates an Ingress test fixture.
func GetIngress() (*v1beta1.Ingress, error) {
	return getIngress("ingress.yaml")
}

// GetIngressComplex creates an Ingress test fixture with multiple backends and path rules.
func GetIngressComplex() (*v1beta1.Ingress, error) {
	return getIngress("ingress-complex.yaml")
}

// GetIngressNamespaced creates 2 Ingress test fixtures in different namespaces.
func GetIngressNamespaced() (*[]v1beta1.Ingress, error) {
	ingr1, err := getIngress("ingress-namespace-1.yaml")
	if err != nil {
		glog.Fatal(err)
	}
	ingr2, err := getIngress("ingress-namespace-2.yaml")
	if err != nil {
		glog.Fatal(err)
	}
	return &[]v1beta1.Ingress{*ingr1, *ingr2}, nil
}

func getIngress(fileName string) (*v1beta1.Ingress, error) {
	ingr, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Print(err)
	}

	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(ingr, nil, nil)
	if err != nil {
		return nil, err
	}
	return obj.(*v1beta1.Ingress), nil
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
			ProvisioningState:       nil,
		},
	}
}

// NewIngressBackendFixture makes a new Ingress Backend for testing
func NewIngressBackendFixture(serviceName string, port int32) *v1beta1.IngressBackend {
	return &v1beta1.IngressBackend{
		ServiceName: serviceName,
		ServicePort: intstr.IntOrString{
			IntVal: port,
		},
	}
}

// NewIngressRuleFixture makes a new Ingress Rule for testing
func NewIngressRuleFixture(host string, urlPath string, be v1beta1.IngressBackend) v1beta1.IngressRule {
	return v1beta1.IngressRule{
		Host: host,
		IngressRuleValue: v1beta1.IngressRuleValue{
			HTTP: &v1beta1.HTTPIngressRuleValue{
				Paths: []v1beta1.HTTPIngressPath{
					{
						Path:    urlPath,
						Backend: be,
					},
				},
			},
		},
	}
}

// NewIngressFixture makes a new Ingress for testing
func NewIngressFixture() *v1beta1.Ingress {
	be80 := NewIngressBackendFixture(ServiceName, 80)
	be443 := NewIngressBackendFixture(ServiceName, 443)

	return &v1beta1.Ingress{
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				NewIngressRuleFixture(Host, URLPath1, *be80),
				NewIngressRuleFixture(Host, URLPath2, *be443),
			},
			TLS: []v1beta1.IngressTLS{
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
					StrVal: containerName,
				},
				Scheme: v1.URISchemeHTTP,
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
					},
					ReadinessProbe: NewProbeFixture(containerName),
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
func NewIngressTestFixture(namespace string, ingressName string) v1beta1.Ingress {
	return v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "hello.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/hi",
									Backend: v1beta1.IngressBackend{
										ServiceName: ServiceName,
										ServicePort: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 80,
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
func NewIngressTestFixtureBasic(namespace string, ingressName string, tls bool) *v1beta1.Ingress {
	ingress := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "hello.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1beta1.IngressBackend{
										ServiceName: ServiceName,
										ServicePort: intstr.IntOrString{
											Type:   intstr.String,
											StrVal: "http",
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
		ingress.Spec.TLS = []v1beta1.IngressTLS{
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
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NameOfSecret,
			Namespace: Namespace,
		},
		StringData: map[string]string{
			"tls.crt": "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNwakNDQVk0Q0NRRDE5NytIcnFITjZ6QU5CZ2txaGtpRzl3MEJBUXNGQURBVk1STXdFUVlEVlFRRERBcHoKWVcxd2JHVXRZWEJ3TUI0WERURTVNRGN6TURJd05Ua3lNRm9YRFRJd01EY3lPVEl3TlRreU1Gb3dGVEVUTUJFRwpBMVVFQXd3S2MyRnRjR3hsTFdGd2NEQ0NBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCCkFOUzFqYzFVdnhLTTVaQ09XRWRTWW0wVkUyNDNIMjkrek5Db1NpVnNFMXN4d1F5bVpuUk9PZVpyMnV5eUlUdmcKblpmZUJCN01qZllkcHJGSGVmR01EUEs4a2xGMGU5bHJNM0dGTjFaaFQwbmFGR2U3SjFHcS85Wk1CMTBXdjllZwpFTTVoMXRmdGdUMGNRbHhFaXJ0a3poQnVISDQycW5pa0U3ZGxsUHMxOTkra3ZudVBQTzBUaEZZQVRCYSt5NUs5Cm1BUmg1U2JyK2JscXJndCt0Zzl0T2RuT2hlK2tyMytuVnBBcGdJZERPSHplVXVrWlpNcU90anZBTGJMRDlmeWYKNlBOMDVXSjBFcDROMWZhZmIzclpEMXZVcnd4OUxxMEpwb0NNeUxzMXVaWGhyWk1IK3dKaUZ5akYzR2dXRDVIUwp6Y2I0K2R1cFFpdWY5WjZNT3Y4d1JDc0NBd0VBQVRBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQUQ3aVBpUGh5CnNhR1FpYTJUUGIvM24yVi83cWVhUUxFYXQ3R3h0UTVDTHpkZXhhL0tnV3RiV0RYTWRneGVTOElOWVpFSUxsck8KaUFtZTRXdTBZeUYrQUdBRjBCVkhxaWJDV1RFb1FmZ2p6S3ZYandpZ1VmWXdhL1ZETWpnYzNybktTbDVvVEhTegpmbE5DMHVqaDk1YWJyNVlDQ1FUdFpUYUh3NU44VmhpamhnUnlDQmtXd2xIZWl0MHVSbEdvS3VIUHh2dVN3djM0CmZ3Uy9GbkcvM0tkc0RtZENDWWYrSm9rbDhHZXFzejNGR3dKWWV1S3pXL0J2UlVxaEdxRlVQVjRNditOVEY2TDkKNXhpRWlnblo1WS9HbjM0aXhWTTRPVldLL0lYUy9vUXFDWkJQWnJDb2pwMnpyMWw4SnVRUU5WQ1lDbEpoajlJTApVNXlteWRBUDVOaStCUT09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K",
			"tls.key": "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV3QUlCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQktvd2dnU21BZ0VBQW9JQkFRRFV0WTNOVkw4U2pPV1EKamxoSFVtSnRGUk51Tng5dmZzelFxRW9sYkJOYk1jRU1wbVowVGpubWE5cnNzaUU3NEoyWDNnUWV6STMySGFheApSM254akF6eXZKSlJkSHZaYXpOeGhUZFdZVTlKMmhSbnV5ZFJxdi9XVEFkZEZyL1hvQkRPWWRiWDdZRTlIRUpjClJJcTdaTTRRYmh4K05xcDRwQk8zWlpUN05mZmZwTDU3anp6dEU0UldBRXdXdnN1U3ZaZ0VZZVVtNi9tNWFxNEwKZnJZUGJUblp6b1h2cEs5L3AxYVFLWUNIUXpoODNsTHBHV1RLanJZN3dDMnl3L1g4bitqemRPVmlkQktlRGRYMgpuMjk2MlE5YjFLOE1mUzZ0Q2FhQWpNaTdOYm1WNGEyVEIvc0NZaGNveGR4b0ZnK1IwczNHK1BuYnFVSXJuL1dlCmpEci9NRVFyQWdNQkFBRUNnZ0VCQU5BM25NU250WmFhRjhwR25RSE1FbzlITjBzSGFKMUMyWWxUZzZsWVB5WmcKOE9IS0xiYWlNS2x2WU5HY21VMjgxV2VwTEExZUhZVVRoMjQ0VXBWeGkrYzlVbG1zRmVSQnZRemQ0OHFKM1F5bApEcDV3Sk5BYi9PNHdaSERxYVFiUktFSnVvZG1qSTRUSG1lb3FLa2ZBS0xzS25wZXFPWHQ0MmRnSDl5dGxxK3ZkCjFyT2J5cHUyTTFZdXlrcEMzcFhBNlovUlJ6dzU1REZXeUF6UGx0aXora0pydWZkQ08xbzlsZTRQK0xkbW43VEEKZkd3a1RBVlBUYVpCelJFOS9ZaUFaeUhzYUJyZnBXUVFoR1I1Z0tIa09Hbm1YM255dm9JWGFmT05pY2szMkZRNwo2bmJXVEJ1RGdyRnBMYmNzMlNxc1JvY25mRnlwSFNYY1BEdlptZDFTeTJFQ2dZRUE3UFBXR3U4VEp1OTFpaENFCk93WlJXNEE2c0lwRmN4UlJ0WTA0Ri9CTTU1Q0lheEhFc3RxOXl2ZmZHZWhMSVVKVzRtakQrVk1Qd0M5QStseGgKT2NOSDZGbUFCV28xNUVOVmVpR29OL1k1dDRIZTFrTS9Helh0clVSczdKOFFqZTduOEZNcDhocnRRcUxEREIzbgpvblhBbWN4eVpranc5cngybGVLSEtNb3dRZHNDZ1lFQTVjN1N2eWFGZFY2bC9LYm5iWkhVOWVUbDlBdFVVeVdvCmhxT2FvMTh5b2tEaXpnYkZ1ekRXaUxnYllkUzRJSkdjc3k3eUEwWUFQNHI2Q1o3NTFiMjBIcDNILzRCT2ZqWnUKWjlydGlCWitpbFdRMFo4MzB6MU54RENYRHJmS1k2eUxsWFFmbEF4WmhjckJVbHA1NUhaVlk3NGlUbHlyMUVoKwpGa29wSVV1V1gvRUNnWUVBNmlvMmhydUpVOHNGZjRHL0M0Mjh6UTQxOGMxVHdOeHR1MXRwK2M1U1VlMjF3d24yCk4wS1FtWXJJQWhSY1d1dnliU0ZYdW9kcFkyWFBjeHZrUVc5SkdzZUlDdEhobkVrbXFlR2xHbGpNeFJzbEd0MnQKK2JnYndFV0UxM3FDbzZGYnVWYVdkMXBBNnI5cXZnaTNwd2R6WlFwMGE3emQxUmgrb0xVVEdTNW03azBDZ1lFQQp6a0VtblRFeHJ2blgrRDdFajI3SHVEcE11UkJDQ3E1TjV5bUhiUUhRTEJnWWQ5bFVOb0hLeXNLU3NPZCtxcHlHCkM3d2lzaFZ0dTVvOGQ4NGJaVjd4L2xpV1hCY0lXL2IyZUhmaG9MYXZzL0RBSGFQTk11WmVtYXNTcUw4RUF4bWwKM2VsdlBpMG5YQkZ5R1R2akVzMnlMZWRFV3hpSmorblFZS2tHNlQ4eUk4RUNnWUVBN0hoVjgzVlhkOWNDb3A5QQpabTJ6OUFrSkxhdUM2eGIvTkZsM25BcllIREF2UnVndlZsSFdUcm5GditscnRUQzZCYVd3WUF2QVBJZllibm5WCjRTcFNVVFlydGI0RWFYdHFVY0tYeVpqV25KaEhFSGtKYXJFNmxDMmxUNWRCTTdmYmdNclIxbUtFaHRyQ3hmYnoKeG1wYWlJb2tZdEx3dkFEZ2RXWkZhWE1DYjM4PQotLS0tLUVORCBQUklWQVRFIEtFWS0tLS0tCg==",
		},
	}
}
