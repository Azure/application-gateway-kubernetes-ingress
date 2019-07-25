// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import "github.com/knative/pkg/apis/istio/v1alpha3"

type istioMatchIdentifier struct {
	Namespace      string
	VirtualService *v1alpha3.VirtualService
	Rule           *v1alpha3.HTTPRoute
	Match          *v1alpha3.HTTPMatchRequest
	Destinations   []*v1alpha3.Destination
	Gateways       []string
}

type istioDestinationIdentifier struct {
	serviceIdentifier
	VirtualService *v1alpha3.VirtualService
	Destination    *v1alpha3.Destination
}
