// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	v1 "k8s.io/api/core/v1"
)

// ShouldProcess determines whether to process an event.
func (c AppGwIngressController) ShouldProcess(event events.Event) bool {
	if pod, ok := event.Value.(*v1.Pod); ok {
		// this pod is not used by any ingress, skip any event for this
		return c.k8sContext.IsPodReferencedByAnyIngress(pod)
	}

	if endpoints, ok := event.Value.(*v1.Endpoints); ok {
		// this pod is not used by any ingress, skip any event for this
		return c.k8sContext.IsEndpointReferencedByAnyIngress(endpoints)
	}

	return true
}
