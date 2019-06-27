package fixtures

import (
	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

const (
	PathFoo = "/foo"
	PathFox = "/fox"
	PathBar = "/bar"
	PathBaz = "/baz"
)

func GetManagedTargets() []*mtv1.AzureIngressManagedTarget {
	return []*mtv1.AzureIngressManagedTarget{
		{
			Spec: mtv1.AzureIngressManagedTargetSpec{
				IP:       IPAddress1,
				Hostname: tests.Host,
				Port:     443,
				Paths: []string{
					PathFoo,
					PathBar,
					PathBaz,
				},
			},
		},
	}

}

func GetProhibitedTargets() []*ptv1.AzureIngressProhibitedTarget {
	return []*ptv1.AzureIngressProhibitedTarget{
		{
			Spec: ptv1.AzureIngressProhibitedTargetSpec{
				IP:       IPAddress1,
				Hostname: tests.Host,
				Port:     443,
				Paths: []string{
					PathFox,
					PathBar,
				},
			},
		},
	}
}
