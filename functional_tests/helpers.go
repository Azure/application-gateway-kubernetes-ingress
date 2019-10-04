// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package functests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/onsi/gomega"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

func check(cbCtx *appgw.ConfigBuilderContext, expectedFilename string, stopChan chan struct{}, ctxt *k8scontext.Context, configBuilder appgw.ConfigBuilder) {
	// Start the informers. This will sync the cache with the latest ingress.
	err := ctxt.Run(stopChan, true, environment.GetFakeEnv())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	appGW, err := configBuilder.Build(cbCtx)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	jsonBlob, err := appGW.MarshalJSON()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	var into map[string]interface{}
	err = json.Unmarshal(jsonBlob, &into)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	jsonBlob, err = json.MarshalIndent(into, "", "    ")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	actualJSONTxt := string(jsonBlob)

	// Repair tests
	// ioutil.WriteFile(expectedFilename, []byte(actualJSONTxt), 0644)

	expectedBytes, err := ioutil.ReadFile(expectedFilename)
	expectedJSON := strings.Trim(string(expectedBytes), "\n")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	linesAct := strings.Split(actualJSONTxt, "\n")
	linesExp := strings.Split(expectedJSON, "\n")

	msg := fmt.Sprintf("Line counts are different: %d vs %d\nActual:\n%s\nExpected:\n%s\nfile: %s", len(linesAct), len(linesExp), actualJSONTxt, expectedJSON, expectedFilename)
	gomega.Expect(len(linesAct)).To(gomega.Equal(len(linesExp)), msg)

	for idx, line := range linesAct {
		curatedLineAct := strings.Trim(line, " ")
		curatedLineExp := strings.Trim(linesExp[idx], " ")
		msg := fmt.Sprintf("Lines at index %d are different:\n%s\nvs expectedJSON:\n%s\nActual JSON:\n%s\nfrom file %s", idx, curatedLineAct, curatedLineExp, actualJSONTxt, expectedFilename)
		gomega.Expect(curatedLineAct).To(gomega.Equal(curatedLineExp), msg)
	}

}
