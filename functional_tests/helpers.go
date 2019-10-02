// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package functional_tests

import (
	"fmt"
	"strings"

	. "github.com/onsi/gomega"
)

func check(jsonTxt string, expected string) {
	linesAct := strings.Split(jsonTxt, "\n")
	linesExp := strings.Split(expected, "\n")

	Expect(len(linesAct)).To(Equal(len(linesExp)), "Line counts are different: ", len(linesAct), " vs ", len(linesExp), "\nActual:", jsonTxt, "\nExpected:", expected)

	for idx, line := range linesAct {
		curatedLineAct := strings.Trim(line, " ")
		curatedLineExp := strings.Trim(linesExp[idx], " ")
		Expect(curatedLineAct).To(Equal(curatedLineExp), fmt.Sprintf("Lines at index %d are different:\n%s\nvs expected:\n%s\nActual JSON:\n%s\n", idx, curatedLineAct, curatedLineExp, jsonTxt))
	}

}
