// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/golang/glog"
)

type portName string
type portsByName map[portName]n.ApplicationGatewayFrontendPort

// GetBlacklistedPorts filters the given list of ports to the list of ports that AGIC is allowed to manage.
func (er ExistingResources) GetBlacklistedPorts() ([]n.ApplicationGatewayFrontendPort, []n.ApplicationGatewayFrontendPort) {

	blacklistedPortSet := er.getBlacklistedPortsSet()

	var nonBlacklistedPorts []n.ApplicationGatewayFrontendPort
	var blacklistedPorts []n.ApplicationGatewayFrontendPort
	for _, port := range er.Ports {
		portJSON, _ := port.MarshalJSON()
		// Is the port associated with a blacklisted listener?
		if _, exists := blacklistedPortSet[portName(*port.Name)]; exists {
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
	uniq := make(portsByName)
	for _, bucket := range portBuckets {
		for _, port := range bucket {
			uniq[portName(*port.Name)] = port
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

	blacklistedSet := indexPortsByName(existingBlacklisted)
	managedSet := indexPortsByName(managedPorts)

	for portName, port := range indexPortsByName(existingNonBlacklisted) {
		_, existsInBlacklist := blacklistedSet[portName]
		_, existsInNewPorts := managedSet[portName]
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

func indexPortsByName(ports []n.ApplicationGatewayFrontendPort) portsByName {
	indexed := make(portsByName)
	for _, port := range ports {
		indexed[portName(*port.Name)] = port
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
