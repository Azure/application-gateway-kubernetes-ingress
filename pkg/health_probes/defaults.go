package health_probes

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

func GetDefaultProbeName(agPrefix string) string {
	return fmt.Sprintf("%sdefaultprobe", agPrefix)
}

func GetDefaultProbe(agPrefix string) n.ApplicationGatewayProbe {
	defProbeName := GetDefaultProbeName(agPrefix)
	defProtocol := n.HTTP
	defHost := "localhost"
	defPath := "/"
	defInterval := int32(30)
	defTimeout := int32(30)
	defUnHealthyCount := int32(3)
	return n.ApplicationGatewayProbe{
		Name: &defProbeName,
		ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
			Protocol:           defProtocol,
			Host:               &defHost,
			Path:               &defPath,
			Interval:           &defInterval,
			Timeout:            &defTimeout,
			UnhealthyThreshold: &defUnHealthyCount,
		},
	}
}
