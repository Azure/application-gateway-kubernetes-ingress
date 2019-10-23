// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package tags

// An App Gateway tag: Resources tagged with this are exclusively managed by a Kubernetes Ingress.
const (
	ManagedByK8sIngress     = "managed-by-k8s-ingress"
	IngressForAKSClusterID  = "ingress-for-aks-cluster-id"
	LastUpdatedByK8sIngress = "last-updated-by-k8s-ingress"
)
