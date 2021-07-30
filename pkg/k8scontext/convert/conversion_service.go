package convert

import (
	multiclusterservice "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/multiclusterservice/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func FromMultiClusterService(gs *multiclusterservice.MultiClusterService) (*v1.Service, bool) {
	if gs == nil {
		return nil, false
	}
	v1Serv := &v1.Service{}
	//copy over metadata
	v1Serv.ObjectMeta = gs.ObjectMeta
	v1Serv.ClusterName = gs.ClusterName
	v1Serv.Labels = gs.Labels
	v1Serv.Annotations = gs.Annotations

	//copy over spec
	v1Serv.Spec.Selector = gs.Spec.Selector.MatchLabels
	for _, port := range gs.Spec.Ports {
		servicePort := v1.ServicePort{}
		servicePort.Name = port.Name
		servicePort.Protocol = v1.Protocol(port.Protocol)
		servicePort.Port = int32(port.Port)
		servicePort.TargetPort = intstr.IntOrString{
			IntVal: int32(port.TargetPort),
		}
		v1Serv.Spec.Ports = append(v1Serv.Spec.Ports, servicePort)
	}

	v1Serv.APIVersion = v1.SchemeGroupVersion.String()
	v1Serv.Kind = "Service"
	return v1Serv, true
}
