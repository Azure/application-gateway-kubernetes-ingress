// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build e2e

package runner

import (
	"fmt"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
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
			_, err = clientset.CoreV1().Namespaces().Create(ns)
			Expect(err).To(BeNil())

			SSLE2ERedirectYamlPath := "testdata/one-namespace-one-ingress/ssl-e2e-redirect/app.yaml"
			klog.Info("Applying yaml: ", SSLE2ERedirectYamlPath)
			err = applyYaml(clientset, namespaceName, SSLE2ERedirectYamlPath)
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
