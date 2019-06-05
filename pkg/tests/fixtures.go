// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package tests

import (
	"fmt"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

// constant values to be used for testing
const (
	Namespace     = "--namespace--" 
	Name          = "--name--" 
	Host          = "bye.com" 
	OtherHost     = "--some-other-hostname--" 
	NameOfSecret  = "--the-name-of-the-secret--" 
	ServiceName   = "--service-name--" 
	NodeName      = "--node-name--" 
	URLPath       = "/healthz" 
	ContainerName = "--container-name--" 
	ContainerPort = int32(9876) 
	ServicePort   = "service-port" 
	SelectorKey   = "app" 
	SelectorValue = "frontend" 
	Subscription  = "--subscription--" 
	ResourceGroup = "--resource-group--" 
	AppGwName     = "--app-gw-name--" 
	IPID1         = "--front-end-ip-id-1--" 
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
				NewIngressRuleFixture(Host, URLPath, *be80),
				NewIngressRuleFixture(Host, URLPath, *be443),
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
				annotations.SslRedirectKey: "true",
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
			Type:   intstr.Int,
			IntVal: ContainerPort,
		},
	}

	httpsPort := v1.ServicePort{
		Name:     "https",
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
				Path: URLPath,
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
						Name:     Name,
						Port:     ContainerPort,
					},
				},
			},
		},
	}
}
