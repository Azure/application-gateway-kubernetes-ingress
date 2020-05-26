// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build e2e

package runner

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

var _ = Describe("LFU", func() {
	var (
		clientset *kubernetes.Clientset
		err       error
	)

	Context("One Namespace One Ingress", func() {
		BeforeEach(func() {
			clientset, err = getClient()
			Expect(err).To(BeNil())
			cleanUp(clientset)
		})

		It("[prohibited-target-test] prohibited service should be available to be accessed", func() {
			// get ip address for 1 ingress
			klog.Info("Getting public IP from blacklisted Ingress...")
			publicIP, _ := getPublicIP(clientset, "test-brownfield-ns-y")
			Expect(publicIP).ToNot(Equal(""))

			// whitlist service will be wiped out by agic
			url_whitelist := fmt.Sprintf("http://%s/x", publicIP)
			_, err = makeGetRequest(url, "brownfield-ns-x.host", 404, true)
			Expect(err).To(BeNil())

			// prohibited service will be kept by agic
			url_blacklist := fmt.Sprintf("http://%s/y", publicIP)
			_, err = makeGetRequest(url, "brownfield-blacklist-ns-y.host", 200, true)
			Expect(err).To(BeNil())

			// delete namespaces for blacklist testing
			for _, nm := range []string{"test-brownfield-ns-x", "test-brownfield-ns-y"} {
				klog.Info("Delete namespaces after blacklist testing: ", nm)
				_, err = clientset.CoreV1().Namespaces().Delete(ns)
				Expect(err).To(BeNil())
			}
		})

		AfterEach(func() {
			// clear all namespaces
			cleanUp(clientset)
		})
	})

})
