// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build unittest

package convert

import (
	"flag"
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/klog/v2"
)

func TestConversion(t *testing.T) {
	klog.InitFlags(nil)
	_ = flag.Set("v", "5")
	_ = flag.Lookup("logtostderr").Value.Set("true")

	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Convert Suite")
}
