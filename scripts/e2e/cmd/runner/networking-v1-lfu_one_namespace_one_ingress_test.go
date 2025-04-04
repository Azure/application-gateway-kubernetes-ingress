// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

//go:build e2e
// +build e2e

package runner

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var _ = Describe("networking-v1-LFU", func() {
	var (
		clientset *kubernetes.Clientset
		crdClient *versioned.Clientset
		err       error
	)

	Context("One Namespace One Ingress", func() {
		BeforeEach(func() {
			clientset, crdClient, err = getClients()
			Expect(err).To(BeNil())

			UseNetworkingV1Ingress = supportsNetworkingV1IngressPackage(clientset)
			skipIfNetworkingV1NotSupport()

			cleanUp(clientset)
		})

		It("[prohibited-target-test] prohibited target should be available to be accessed", func() {
			namespaceName := "e2e-prohibited-target"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace: ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			appYamlPath := "testdata/networking-v1/one-namespace-one-ingress/prohibited-target/app.yaml"
			klog.Info("Applying yaml: ", appYamlPath)
			err = applyYaml(clientset, crdClient, namespaceName, appYamlPath)
			Expect(err).To(BeNil())
			time.Sleep(30 * time.Second)

			// get ip address for 1 ingress
			klog.Info("Getting public IP of the app gateway")
			publicIP, err := getAzurePublicIP()
			Expect(err).To(BeNil())

			klog.Infof("Public IP: %s", publicIP)

			protectedPath := fmt.Sprintf("http://%s/landing/", publicIP)
			_, err = makeGetRequest(protectedPath, "www.microsoft.com", 302, true)
			Expect(err).To(BeNil())

			ingressPath := fmt.Sprintf("http://%s/aspnet", publicIP)
			_, err = makeGetRequest(ingressPath, "www.microsoft.com", 200, true)
			Expect(err).To(BeNil())

			klog.Info("Deleting yaml: ", appYamlPath)
			err = deleteYaml(clientset, crdClient, namespaceName, appYamlPath)
			Expect(err).To(BeNil())
			time.Sleep(30 * time.Second)

			_, err = makeGetRequest(protectedPath, "www.microsoft.com", 302, true)
			Expect(err).To(BeNil())

			_, err = makeGetRequest(ingressPath, "www.microsoft.com", 502, true)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			// clear all namespaces
			cleanUp(clientset)
		})
	})
})
