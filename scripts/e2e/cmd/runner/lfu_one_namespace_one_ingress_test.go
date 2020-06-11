// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build e2e

package runner

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			publicIP, _ := getPublicIP(clientset, "test-brownfield-ns")
			Expect(publicIP).ToNot(Equal(""))

			//prohibited service will be kept by agic
			url_blacklist := fmt.Sprintf("http://%s/blacklist", publicIP)
			_, err = makeGetRequest(url_blacklist, "brownfield-blacklist-ns.host", 200, true)
			Expect(err).To(BeNil())

			//delete namespaces for blacklist testing
			deleteOptions := &metav1.DeleteOptions{
				GracePeriodSeconds: to.Int64Ptr(0),
			}

			klog.Info("Delete namespaces test-brownfield-ns after blacklist testing...")
			err = clientset.CoreV1().Namespaces().Delete("test-brownfield-ns", deleteOptions)
			Expect(err).To(BeNil())
		})
	})

})
