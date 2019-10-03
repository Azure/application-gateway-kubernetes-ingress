// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package functional_tests

import (
	"encoding/json"

	. "github.com/onsi/gomega"

	. "github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

func two_ingresses(ctxt *k8scontext.Context, stopChannel chan struct{}, cbCtx *ConfigBuilderContext, configBuilder ConfigBuilder) {
	// Start the informers. This will sync the cache with the latest ingress.
	err := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
	Expect(err).ToNot(HaveOccurred())

	appGW, err := configBuilder.Build(cbCtx)
	Expect(err).ToNot(HaveOccurred())

	jsonBlob, err := appGW.MarshalJSON()
	Expect(err).ToNot(HaveOccurred())

	var into map[string]interface{}
	err = json.Unmarshal(jsonBlob, &into)
	Expect(err).ToNot(HaveOccurred())

	jsonBlob, err = json.MarshalIndent(into, "", "    ")
	Expect(err).ToNot(HaveOccurred())

	actualJsonTxt := string(jsonBlob)

	check(actualJsonTxt, "two_ingresses.json")
}
