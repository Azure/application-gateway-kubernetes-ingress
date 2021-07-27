// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package v1alpha1

import (
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiClusterIngress is the resource AGIC is watching in MultiCluster mode
type MultiClusterIngress struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MultiClusterIngressSpec `json:"spec"`

	Status v1.IngressStatus `json:"status"`
}

// MultiClusterIngressSpec is the Spec for MultiClusterIngress Resource
type MultiClusterIngressSpec struct {
	IngressSpec v1.IngressSpec `json:"ingressSpec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiClusterIngressList is the list of MultiCluster Ingresses
type MultiClusterIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MultiClusterIngress `json:"items"`
}
