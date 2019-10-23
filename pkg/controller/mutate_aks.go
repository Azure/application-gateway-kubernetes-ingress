// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import "github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"

func (c AppGwIngressController) MutateAKS(event events.Event) error {
	return nil
}
