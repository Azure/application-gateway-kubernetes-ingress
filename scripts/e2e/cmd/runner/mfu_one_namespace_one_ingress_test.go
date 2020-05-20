// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build e2e

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

var _ = Describe("MFU", func() {
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

		It("[ssl-e2e-redirect] ssl termination and ssl redirect to https backend should work", func() {
			namespaceName := "e2e-ssl-e2e-redirect"
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

			// get ip address for 1 ingress
			klog.Info("Getting public IP from Ingress...")
			publicIP, _ := getPublicIP(clientset, namespaceName)
			Expect(publicIP).ToNot(Equal(""))

			urlHttp := fmt.Sprintf("http://%s/index.html", publicIP)
			urlHttps := fmt.Sprintf("https://%s/index.html", publicIP)
			// http get to return 200 ok
			resp, err := makeGetRequest(urlHttp, "", 301, true)
			Expect(err).To(BeNil())
			redirectLocation := resp.Header.Get("Location")
			klog.Infof("redirect location: %s", redirectLocation)
			Expect(redirectLocation).To(Equal(urlHttps))
			// https get to return 200 ok
			_, err = makeGetRequest(urlHttps, "", 200, true)
			Expect(err).To(BeNil())
		})

		It("[three-namespaces] containers with the same probe and labels in 3 different namespaces should have unique and working health probes", func() {
			// http get to return 200 ok
			for _, nm := range []string{"e2e-ns-x", "e2e-ns-y", "e2e-ns-z"} {
				ns := &v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: nm,
					},
				}
				klog.Info("Creating namespace: ", nm)
				_, err = clientset.CoreV1().Namespaces().Create(ns)
				Expect(err).To(BeNil())
			}
			threeNamespacesYamlPath := "testdata/one-namespace-one-ingress/three-namespaces/app.yaml"
			klog.Info("Applying yaml: ", threeNamespacesYamlPath)
			err = applyYaml(clientset, "", threeNamespacesYamlPath)
			Expect(err).To(BeNil())
			time.Sleep(30 * time.Second)

			// get ip address for 1 ingress
			klog.Info("Getting public IP from Ingress...")
			publicIP, _ := getPublicIP(clientset, "e2e-ns-x")
			Expect(publicIP).ToNot(Equal(""))

			hosts := []string{"ws-e2e-ns-x.mis.li", "ws-e2e-ns-y.mis.li", "ws-e2e-ns-z.mis.li"}
			url := fmt.Sprintf("http://%s/status/200", publicIP)
			for _, host := range hosts {
				_, err = makeGetRequest(url, host, 200, true)
				Expect(err).To(BeNil())
			}
		})

		AfterEach(func() {
			// clear all namespaces
			cleanUp(clientset)
		})
	})
})
