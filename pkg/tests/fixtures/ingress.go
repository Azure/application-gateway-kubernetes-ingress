package fixtures

import (
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// GetIngress creates an Ingress struct.
func GetIngress() *networking.Ingress {
	return &networking.Ingress{
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "foo.baz",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: "/",
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: tests.ServiceName,
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
			TLS: []networking.IngressTLS{
				{
					Hosts: []string{
						"www.contoso.com",
						"ftp.contoso.com",
						tests.Host,
						"",
					},
					SecretName: tests.NameOfSecret,
				},
				{
					Hosts:      []string{},
					SecretName: tests.NameOfSecret,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotations.IngressClassKey: tests.IngressClassController,
				annotations.SslRedirectKey:  "true",
			},
			Namespace: tests.Namespace,
			Name:      tests.Name,
		},
	}
}

// GetIngressWithProhibitedTargetConflict returns ingress with /foo and /fox as paths
func GetIngressWithProhibitedTargetConflict() *networking.Ingress {
	return &networking.Ingress{
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					// Rule with no Paths
					Host: tests.OtherHost,
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{},
					},
				},
				{
					// Rule with Paths
					Host: tests.Host,
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: PathFoo,
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: tests.ServiceName,
											Port: networking.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
								{
									Path: PathFox,
									Backend: networking.IngressBackend{
										Service: &networking.IngressServiceBackend{
											Name: tests.ServiceName,
											Port: networking.ServiceBackendPort{
												Number: 443,
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
