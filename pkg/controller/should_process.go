// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

// ShouldProcess determines whether to process an event.
func (c AppGwIngressController) ShouldProcess(event events.Event) (bool, *string) {
	if pod, ok := event.Value.(*v1.Pod); ok {
		// this pod is not used by any ingress, skip any event for this
		reason := fmt.Sprintf("pod %s/%s is not used by any Ingress", pod.Namespace, pod.Name)
		return c.k8sContext.IsPodReferencedByAnyIngress(pod), to.StringPtr(reason)
	}

	if endpoints, ok := event.Value.(*v1.Endpoints); ok {
		if endpoints.Namespace == "default" && endpoints.Name == "aad-pod-identity-mic" {
			// Ignore AAD Pod Identity
			return false, nil
		}

		// this pod is not used by any ingress, skip any event for this
		reason := fmt.Sprintf("endpoint %s/%s is not used by any Ingress", endpoints.Namespace, endpoints.Name)
		return c.k8sContext.IsEndpointReferencedByAnyIngress(endpoints), to.StringPtr(reason)
	}

	if event.Type == events.PeriodicReconcile {
		appGw, _, err := c.GetAppGw()
		if err != nil {
			glog.Error("Error Retrieving AppGw for k8s event. ", err)
			reason := err.Error()
			return false, to.StringPtr(reason)
		}

		glog.V(5).Info("Triggered by reconciler event")
		if c.configIsSame(appGw) {
			reason := "NoOp: current gateway state == cached gateway state"
			glog.V(5).Info(reason)
			return false, to.StringPtr(reason)
		}
	}

	return true, nil
}
