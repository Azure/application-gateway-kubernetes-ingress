// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiClusterService is the Schema for the multi cluster services API
type MultiClusterService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MultiClusterServiceSpec   `json:"spec,omitempty"`
	Status MultiClusterServiceStatus `json:"status,omitempty"`
}

// MultiClusterServiceSpec defines the desired state of MultiClusterService
type MultiClusterServiceSpec struct {
	// LabelSelector for the multi cluster service. Services with the same name of MultiClusterService would be selected if the selector is not set.
	Selector metav1.LabelSelector `json:"selector,omitempty"`
	// Ports for the multi cluster service.
	Ports []MultiClusterServicePort `json:"ports,omitempty"`
	// ClusterSet for the multi cluster service.
	ClusterSet string `json:"clusterSet,omitempty"`
}

// MultiClusterServicePort defines the spec for MultiClusterService port
type MultiClusterServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int    `json:"port,omitempty"`
	TargetPort int    `json:"targetPort,omitempty"`
}

// MultiClusterServiceStatus defines the observed state of MultiClusterService
type MultiClusterServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Endpoints represents a list of endpoint for the multi cluster service.
	Endpoints []MultiClusterEndpoint `json:"endpoints,omitempty"`
	VIP       string                 `json:"vip,omitempty"`
	State     string                 `json:"state,omitempty"`
}

// MultiClusterEndpoint defines the endpoints for the multi cluster service.
type MultiClusterEndpoint struct {
	Cluster   string   `json:"cluster,omitempty"`
	Service   string   `json:"service,omitempty"`
	IP        string   `json:"ip,omitempty"`
	Endpoints []string `json:"endpoints,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MultiClusterServiceList contains a list of MultiClusterService
type MultiClusterServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MultiClusterService `json:"items"`
}
