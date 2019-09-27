// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

var _ = Describe("test NewAppGwIngressController", func() {

	Context("ensure NewAppGwIngressController works as expected", func() {

		appGwClient := n.ApplicationGatewaysClient{}
		appGwIdentifier := appgw.Identifier{}
		k8sContext := &k8scontext.Context{}
		recorder := record.NewFakeRecorder(0)
		controller := NewAppGwIngressController(appGwClient, appGwIdentifier, k8sContext, recorder, nil)
		It("should have created the AppGwIngressController struct", func() {
			Expect(controller.appGwClient.Client.SkipResourceProviderRegistration).To(BeFalse())
			err := controller.Start(environment.GetEnv())
			Expect(err).To(HaveOccurred())
			controller.Stop()
		})
	})
})
