// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build e2eingressclass

package runner

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

var _ = Describe("IngressClass", func() {
	var (
		clientset *kubernetes.Clientset
		err       error
	)

	Context("Test Ingress Class", func() {
		BeforeEach(func() {
			clientset, err = getClient()
			Expect(err).To(BeNil())

			cleanUp(clientset)
		})

		It("[ingress-class] should only pick up ingresses that match the class specified in helm", func() {
			namespaceName := "e2e-ingress-class"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace: ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(ns)
			Expect(err).To(BeNil())

			// This application has two ingresses:
			// 1. One Ingress: uses "azure/application-gateway" as ingress class
			// 2. Two Ingress: uses "custom-ingress-class" as ingress class
			// We expect that AGIC will use ingress with "custom-ingress-class"
			SSLIngressClassYamlPath := "testdata/ingress-class/app.yaml"
			klog.Info("Applying yaml: ", SSLIngressClassYamlPath)
			err = applyYaml(clientset, namespaceName, SSLIngressClassYamlPath)
			Expect(err).To(BeNil())
			time.Sleep(30 * time.Second)

			// get ip address for 1 ingress
			klog.Info("Getting public IP from Ingress...")
			publicIP, _ := getPublicIP(clientset, namespaceName)
			Expect(publicIP).ToNot(Equal(""))

			url := fmt.Sprintf("http://%s/status/200", publicIP)

			// should return 404 as this ingress is using "azure/application-gateway"
			_, err = makeGetRequest(url, "www.default.com", 404, true)
			Expect(err).To(BeNil(), "This should return 404 as this ingress is using 'azure/application-gateway'")

			// should return 200 as this ingress is using "custom-ingress-class"
			_, err = makeGetRequest(url, "www.custom.com", 200, true)
			Expect(err).To(BeNil(), "This should return 200 as this ingress is using 'custom-ingress-class'")
		})

		AfterEach(func() {
			// clear all namespaces
			cleanUp(clientset)
		})
	})
})
