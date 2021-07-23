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

// GlobalService is the Schema for the globalservices API
type GlobalService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GlobalServiceSpec   `json:"spec,omitempty"`
	Status GlobalServiceStatus `json:"status,omitempty"`
}

// GlobalServiceSpec defines the desired state of GlobalService
type GlobalServiceSpec struct {
	// LabelSelector for the global service. Services with the same name of GlobalService would be selected if the selector is not set.
	Selector metav1.LabelSelector `json:"selector,omitempty"`
	// Ports for the global service.
	Ports []GlobalServicePort `json:"ports,omitempty"`
	// ClusterSet for the global service.
	ClusterSet string `json:"clusterSet,omitempty"`
}

// GlobalServicePort defines the spec for GlobalService port
type GlobalServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int    `json:"port,omitempty"`
	TargetPort int    `json:"targetPort,omitempty"`
}

// GlobalServiceStatus defines the observed state of GlobalService
type GlobalServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Endpoints represents a list of endpoint for the global service.
	Endpoints []GlobalEndpoint `json:"endpoints,omitempty"`
	VIP       string           `json:"vip,omitempty"`
	State     string           `json:"state,omitempty"`
}

// GlobalEndpoint defines the endpoints for the global service.
type GlobalEndpoint struct {
	Cluster   string   `json:"cluster,omitempty"`
	Service   string   `json:"service,omitempty"`
	IP        string   `json:"ip,omitempty"`
	Endpoints []string `json:"endpoints,omitempty"`
}

// GlobalServiceList contains a list of GlobalService
type GlobalServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GlobalService `json:"items"`
}
