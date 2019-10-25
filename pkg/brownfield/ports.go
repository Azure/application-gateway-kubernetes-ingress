// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"strings"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/golang/glog"
)

type portName string
type portsByPortNumber map[int32]n.ApplicationGatewayFrontendPort

// GetBlacklistedPorts filters the given list of ports to the list of ports that AGIC is allowed to manage.
func (er ExistingResources) GetBlacklistedPorts() ([]n.ApplicationGatewayFrontendPort, []n.ApplicationGatewayFrontendPort) {

	blacklistedPortSet := er.getBlacklistedPortsSet()

	var nonBlacklistedPorts []n.ApplicationGatewayFrontendPort
	var blacklistedPorts []n.ApplicationGatewayFrontendPort
	for _, port := range er.Ports {
		portJSON, _ := port.MarshalJSON()
		// Is the port associated with a blacklisted listener?
		if _, isBlacklisted := blacklistedPortSet[portName(*port.Name)]; isBlacklisted {
			glog.V(5).Infof("[brownfield] Port %s is blacklisted: %s", portName(*port.Name), portJSON)
			blacklistedPorts = append(blacklistedPorts, port)
			continue
		}
		glog.V(5).Infof("[brownfield] Port %s is not blacklisted: %s", portName(*port.Name), portJSON)
		nonBlacklistedPorts = append(nonBlacklistedPorts, port)
	}
	return blacklistedPorts, nonBlacklistedPorts
}

// MergePorts merges list of lists of ports into a single list, maintaining uniqueness.
func MergePorts(portBuckets ...[]n.ApplicationGatewayFrontendPort) []n.ApplicationGatewayFrontendPort {
	uniq := make(portsByPortNumber)
	for _, bucket := range portBuckets {
		for _, port := range bucket {
			// Add the port from the list only when it is missing, otherwise use the existing one.
			if _, exists := uniq[*port.Port]; !exists {
				uniq[*port.Port] = port
			}
		}
	}
	var merged []n.ApplicationGatewayFrontendPort
	for _, port := range uniq {
		merged = append(merged, port)
	}
	return merged
}

// LogPorts emits a few log lines detailing what Ports are created, blacklisted, and removed from ARM.
func LogPorts(existingBlacklisted []n.ApplicationGatewayFrontendPort, existingNonBlacklisted []n.ApplicationGatewayFrontendPort, managedPorts []n.ApplicationGatewayFrontendPort) {
	var garbage []n.ApplicationGatewayFrontendPort

	blacklistedSet := indexPortsByPortNumber(existingBlacklisted)
	managedSet := indexPortsByPortNumber(managedPorts)

	for portNumber, port := range indexPortsByPortNumber(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[portNumber]
		_, existsInNewPorts := managedSet[portNumber]
		if !existsInBlacklist && !existsInNewPorts {
			garbage = append(garbage, port)
		}
	}

	glog.V(3).Info("[brownfield] Ports AGIC created: ", getPortNames(managedPorts))
	glog.V(3).Info("[brownfield] Existing Blacklisted Ports AGIC will retain: ", getPortNames(existingBlacklisted))
	glog.V(3).Info("[brownfield] Existing Ports AGIC will remove: ", getPortNames(garbage))
}

func getPortNames(port []n.ApplicationGatewayFrontendPort) string {
	var names []string
	for _, p := range port {
		names = append(names, *p.Name)
	}
	if len(names) == 0 {
		return "n/a"
	}
	return strings.Join(names, ", ")
}

func indexPortsByPortNumber(ports []n.ApplicationGatewayFrontendPort) portsByPortNumber {
	indexed := make(portsByPortNumber)
	for _, port := range ports {
		indexed[*port.Port] = port
	}
	return indexed
}

func (er ExistingResources) getBlacklistedPortsSet() map[portName]interface{} {
	// Get the list of blacklisted listeners, from which we can determine what ports should be blacklisted.
	existingBlacklistedListeners, _ := er.GetBlacklistedListeners()
	blacklistedPortSet := make(map[portName]interface{})
	for _, listener := range existingBlacklistedListeners {
		if listener.FrontendPort != nil && listener.FrontendPort.ID != nil {
			portName := portName(utils.GetLastChunkOfSlashed(*listener.FrontendPort.ID))
			blacklistedPortSet[portName] = nil
		}
	}
	return blacklistedPortSet
}
