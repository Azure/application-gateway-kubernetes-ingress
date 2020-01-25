// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
)

var _ = Describe("test NewAppGwIngressController", func() {

	Context("ensure NewAppGwIngressController works as expected", func() {

		azClient := azure.NewFakeAzClient()
		appGwIdentifier := appgw.Identifier{}
		k8sContext := &k8scontext.Context{}
		recorder := record.NewFakeRecorder(0)
		controller := NewAppGwIngressController(azClient, appGwIdentifier, k8sContext, recorder, metricstore.NewFakeMetricStore(), nil, false)
		It("should have created the AppGwIngressController struct", func() {
			err := controller.Start(environment.GetEnv())
			Expect(err).To(HaveOccurred())
			controller.Stop()
		})
	})
})
