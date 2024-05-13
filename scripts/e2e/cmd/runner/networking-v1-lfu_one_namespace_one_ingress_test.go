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
	"strings"
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
			ip, err := getPublicIPAddress()
			Expect(err).To(BeNil())

			publicIP := *ip.IPAddress
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

		It("[sub-resource-prefix] should be use the sub-resource-prefix to prefix sub-resources", func() {
			env := GetEnv()
			klog.Infof("'subResourceNamePrefix': %s", env.SubResourceNamePrefix)
			Expect(env.SubResourceNamePrefix).ToNot(Equal(""), "Please make sure that environment variable 'subResourceNamePrefix' is set")

			namespaceName := "e2e-sub-resource-prefix"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace: ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			SSLE2ERedirectYamlPath := "testdata/networking-v1/one-namespace-one-ingress/ssl-e2e-redirect/app.yaml"
			klog.Info("Applying yaml: ", SSLE2ERedirectYamlPath)
			err = applyYaml(clientset, crdClient, namespaceName, SSLE2ERedirectYamlPath)
			Expect(err).To(BeNil())
			time.Sleep(30 * time.Second)

			gateway, err := getGateway()
			Expect(err).To(BeNil())

			prefixUsed := false
			for _, listener := range *gateway.HTTPListeners {
				klog.Infof("checking listener %s for %s", *listener.Name, env.SubResourceNamePrefix)
				if strings.HasPrefix(*listener.Name, env.SubResourceNamePrefix) {
					klog.Infof("found %s that uses the prefix", *listener.Name)
					prefixUsed = true
					break
				}
			}

			Expect(prefixUsed).To(BeTrue(), "%s wasn't used for naming the sub-resource of app gateway. Currently, this check looks at HTTP listener only", env.SubResourceNamePrefix)
		})

		AfterEach(func() {
			// clear all namespaces
			cleanUp(clientset)
		})
	})
})
