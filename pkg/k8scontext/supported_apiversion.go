// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var (
	runtimeScheme = k8sruntime.NewScheme()

	// IsNetworkingV1PackageSupported is flag that indicates whether networking/v1beta ingress should be used instead.
	IsNetworkingV1PackageSupported bool

	// IsInMultiClusterMode is a flag to indicate that AGIC should monitor MutliCluster CRDs
	IsInMultiClusterMode bool
)

func init() {
	extensionsv1beta1.AddToScheme(runtimeScheme)
	networkingv1.AddToScheme(runtimeScheme)
}

// SupportsNetworkingPackage checks if the package "k8s.io/api/networking/v1"
// is available or not and if Ingress V1 is supported (k8s >= v1.19.0)
// https://kubernetes.io/blog/2021/07/14/upcoming-changes-in-kubernetes-1-22/#what-to-do
func SupportsNetworkingPackage(client clientset.Interface) bool {
	// check kubernetes version to use new ingress package or not
	version119, _ := version.ParseGeneric("v1.19.0")

	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		return false
	}

	runningVersion, err := version.ParseGeneric(serverVersion.String())
	if err != nil {
		klog.Errorf("unexpected error parsing running Kubernetes version: %v", err)
		return false
	}

	return runningVersion.AtLeast(version119)
}
