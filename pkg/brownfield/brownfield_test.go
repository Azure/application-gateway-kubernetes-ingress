// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package brownfield

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAppgw(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Brownfield Deployment Tests")
}
