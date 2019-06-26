package brownfield

import (
	"reflect"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"

	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
)

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

// GetProhibitedTargetList returns the list of Targets given a list ProhibitedTarget CRDs.
func GetProhibitedTargetList(prohibitedTargets []*ptv1.AzureIngressProhibitedTarget) *[]Target {
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
				Path:     to.StringPtr(path),
			})
		}
	}
	return &target
}

// GetManagedTargetList returns the list of Targets given a list ManagedTarget CRDs.
func GetManagedTargetList(managedTarget []*mtv1.AzureIngressManagedTarget) *[]Target {
	var target []Target
	for _, managedTarget := range managedTarget {
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
				Path:     to.StringPtr(path),
			})
		}
	}
	return &target
}

// TODO(draychev)
func normalizePath(path string) string {
	trimmed, prevTrimmed := "", path
	cutset := "*/"
	for trimmed != prevTrimmed {
		prevTrimmed = trimmed
		trimmed = strings.TrimRight(path, cutset)
	}
	return trimmed
}
