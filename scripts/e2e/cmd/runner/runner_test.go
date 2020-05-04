// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build e2e

package runner

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

func TestMFU(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("report.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Run E2E MFU Test suite", []Reporter{junitReporter})
}

var _ = Describe("Most frequenty run test suite", func() {
	Context("one namespace many ingresses", func() {
		var clientset *kubernetes.Clientset
		var namespaceName string
		var err error

		BeforeEach(func() {
			clientset, err = getClient()
			Expect(err).To(BeNil())

			// clear all namespaces
			cleanUp(clientset)

			// create namespace
			namespaceName = "e2e-manyingresses"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(ns)
			Expect(err).To(BeNil())

			// create objects in the yaml
			path := "testdata/same-namespace-many-ingress/generated.yaml"
			klog.Info("Applying yaml ", path)
			err := applyYaml(clientset, namespaceName, path)
			Expect(err).To(BeNil())

			time.Sleep(30 * time.Second)
		})

		It("should have get 200 for each ingress", func() {
			// get ip address for 1 ingress
			klog.Info("Getting public IP from Ingress...")
			publicIP, err := getPublicIP(clientset, namespaceName)
			Expect(err).To(BeNil())
			Expect(publicIP).ToNot(Equal(""))

			// make curl request
			hosts := []string{"appa.", "appb."}
			url := fmt.Sprintf("https://%s/status/200", publicIP)
			klog.Infof("Sending get request %s ...", url)
			for i := 1; i <= 50; i++ {
				for _, host := range hosts {
					hostIndex := host + strconv.Itoa(i)
					klog.Infof("Sending request with host %s ...", hostIndex)
					err = makeGetRequest(url, hostIndex, 200)
					Expect(err).To(BeNil())
				}
			}
		})

		AfterEach(func() {
			// clear all namespaces
			cleanUp(clientset)
		})
	})
})
