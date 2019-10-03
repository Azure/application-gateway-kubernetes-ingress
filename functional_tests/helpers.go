// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package functests

import (
	"fmt"
	"io/ioutil"
	"strings"

	. "github.com/onsi/gomega"
)

func check(jsonTxt string, expected_filename string) {
	expectedBytes, err := ioutil.ReadFile(expected_filename)
	expectedJSON := strings.Trim(string(expectedBytes), "\n")
	Expect(err).ToNot(HaveOccurred())

	linesAct := strings.Split(jsonTxt, "\n")
	linesExp := strings.Split(expectedJSON, "\n")

	Expect(len(linesAct)).To(Equal(len(linesExp)), "Line counts are different: ", len(linesAct), " vs ", len(linesExp), "\nActual:", jsonTxt, "\nExpectedJSON:", expectedJSON)

	for idx, line := range linesAct {
		curatedLineAct := strings.Trim(line, " ")
		curatedLineExp := strings.Trim(linesExp[idx], " ")
		Expect(curatedLineAct).To(Equal(curatedLineExp), fmt.Sprintf("Lines at index %d are different:\n%s\nvs expectedJSON:\n%s\nActual JSON:\n%s\n", idx, curatedLineAct, curatedLineExp, jsonTxt))
	}

}
