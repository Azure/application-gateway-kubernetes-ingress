// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"encoding/json"
	"github.com/golang/glog"
	"reflect"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

type NameToTarget map[string]Target

type ListenersByName map[string]*n.ApplicationGatewayHTTPListener

type URLPathMapByName map[string]n.ApplicationGatewayURLPathMap

// Target uniquely identifies a subset of App Gateway configuration, which AGIC will manage or be prohibited from managing.
type Target struct {
	Hostname string
	Port     int32
	Path     *string
}

// IsIn figures out whether a given Target objects in a list of Target objects.
func (t Target) IsIn(targetList *[]Target) bool {
	for _, otherTarget := range *targetList {
		if reflect.DeepEqual(t, otherTarget) {
			// Found it
			return true
		}
	}

	// Did not find it
	return false
}

// prettyTarget is used for pretty-printing the Target struct for debugging purposes.
type prettyTarget struct {
	Hostname string `json:"Hostname"`
	Port     int32  `json:"Port"`
	Path     string `json:"Path,omitempty"`
}

// MarshalJSON converts the Target object to a JSON byte array.
func (t Target) MarshalJSON() []byte {
	pt := prettyTarget{
		Hostname: t.Hostname,
		Port:     t.Port,
	}
	if t.Path != nil {
		pt.Path = *t.Path
	}
	jsonBytes, err := json.Marshal(pt)
	if err != nil {
		glog.Error("Failed Marshaling Target object:", err)
	}
	return jsonBytes
}

// getProhibitedTargetList returns the list of Targets given a list ProhibitedTarget CRDs.
func getProhibitedTargetList(prohibitedTargets []*ptv1.AzureIngressProhibitedTarget) *[]Target {
	var target []Target
	for _, prohibitedTarget := range prohibitedTargets {
		if len(prohibitedTarget.Spec.Paths) == 0 {
			target = append(target, Target{
				Hostname: prohibitedTarget.Spec.Hostname,
				Port:     prohibitedTarget.Spec.Port,
				Path:     nil,
			})
		}
		for _, path := range prohibitedTarget.Spec.Paths {
			target = append(target, Target{
				Hostname: prohibitedTarget.Spec.Hostname,
				Port:     prohibitedTarget.Spec.Port,
				Path:     to.StringPtr(normalizePath(path)),
			})
		}
	}
	return &target
}

// getManagedTargetList returns the list of Targets given a list ManagedTarget CRDs.
func getManagedTargetList(managedTargets []*mtv1.AzureIngressManagedTarget) *[]Target {
	var target []Target
	for _, managedTarget := range managedTargets {
		if len(managedTarget.Spec.Paths) == 0 {
			target = append(target, Target{
				Hostname: managedTarget.Spec.Hostname,
				Port:     managedTarget.Spec.Port,
				Path:     nil,
			})
		}
		for _, path := range managedTarget.Spec.Paths {
			target = append(target, Target{
				Hostname: managedTarget.Spec.Hostname,
				Port:     managedTarget.Spec.Port,
				Path:     to.StringPtr(normalizePath(path)),
			})
		}
	}
	return &target
}

func normalizePath(path string) string {
	trimmed, prevTrimmed := "", path
	cutset := "*/"
	for trimmed != prevTrimmed {
		prevTrimmed = trimmed
		trimmed = strings.TrimRight(path, cutset)
	}
	return trimmed
}
