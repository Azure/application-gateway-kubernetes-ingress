package appgw

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) getFrontendPorts(ingressList []*v1beta1.Ingress) *[]network.ApplicationGatewayFrontendPort {
	allPorts := make(map[int32]interface{})
	for _, ingress := range ingressList {
		fePorts, _ := builder.processIngressRules(ingress)
		for _, port := range fePorts.ToSlice() {
			allPorts[port.(int32)] = nil
		}
	}

	// fallback to default listener as placeholder if no listener is available
	if len(allPorts) == 0 {
		port := defaultFrontendListenerIdentifier().FrontendPort
		allPorts[port] = nil
	}

	var frontendPorts []network.ApplicationGatewayFrontendPort
	for port := range allPorts {
		frontendPortName := generateFrontendPortName(port)
		frontendPorts = append(frontendPorts, network.ApplicationGatewayFrontendPort{
			Etag: to.StringPtr("*"),
			Name: &frontendPortName,
			ApplicationGatewayFrontendPortPropertiesFormat: &network.ApplicationGatewayFrontendPortPropertiesFormat{
				Port: to.Int32Ptr(port),
			},
		})
	}
	return &frontendPorts
}
