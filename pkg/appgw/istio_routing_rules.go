// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	//"sort"
	//"strconv"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	//"github.com/golang/glog"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	//"k8s.io/api/extensions/v1beta1"
	//"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	//"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	//"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
)

func (c *appGwConfigBuilder) getIstioPathMap(cbCtx *ConfigBuilderContext, listenerID listenerIdentifier, listenerAzConfig listenerAzConfig, virtualService *v1alpha3.VirtualService, rule *v1alpha3.HTTPRoute) *n.ApplicationGatewayURLPathMap {
	pathMap := n.ApplicationGatewayURLPathMap{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(generateURLPathMapName(listenerID)),
		ID:   to.StringPtr(c.appGwIdentifier.urlPathMapID(generateURLPathMapName(listenerID))),
		ApplicationGatewayURLPathMapPropertiesFormat: &n.ApplicationGatewayURLPathMapPropertiesFormat{},
	}

	/* TODO(rhea): add defaults and path rules */
	return &pathMap
}
