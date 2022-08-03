// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build unittest

package fixtures

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIngressTestFixtureFactories(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ingress Test Fixture Factories Suite")
}
