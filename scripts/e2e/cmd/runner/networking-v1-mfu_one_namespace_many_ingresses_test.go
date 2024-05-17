// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

//go:build e2e
// +build e2e

package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var _ = Describe("networking-v1-MFU", func() {
	var (
		clientset *kubernetes.Clientset
		crdClient *versioned.Clientset
		err       error
	)

	Context("One Namespace Many Ingresses", func() {
		var namespaceName string

		BeforeEach(func() {
			clientset, crdClient, err = getClients()
			Expect(err).To(BeNil())

			UseNetworkingV1Ingress = supportsNetworkingV1IngressPackage(clientset)
			skipIfNetworkingV1NotSupport()

			cleanUp(clientset)
		})

		It("[three-ingresses-slash-sth] path based routing with backend-path-prefix should work", func() {
			// create namespace
			namespaceName = "e2e-three-ings"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace: ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// create objects in the yaml
			path := "testdata/networking-v1/one-namespace-many-ingresses/three-ingresses-slash-sth/app.yaml"
			klog.Info("Applying yaml: ", path)
			err = applyYaml(clientset, crdClient, namespaceName, path)
			Expect(err).To(BeNil())

			time.Sleep(30 * time.Second)

			// get ip address for 1 ingress
			klog.Info("Getting public IP from Ingress...")
			publicIP, err := getPublicIP(clientset, namespaceName)
			Expect(err).To(BeNil())
			Expect(publicIP).ToNot(Equal(""))

			var url string
			url = fmt.Sprintf("http://%s", publicIP)
			_, err = makeGetRequest(url, "ws.mis.li", 200, true)
			Expect(err).To(BeNil())

			url = fmt.Sprintf("http://%s/igloo", publicIP)
			_, err = makeGetRequest(url, "ws.mis.li", 200, true)
			Expect(err).To(BeNil())

			url = fmt.Sprintf("http://%s/kuard", publicIP)
			_, err = makeGetRequest(url, "ws.mis.li", 200, true)
			Expect(err).To(BeNil())

			url = fmt.Sprintf("http://%s/fail", publicIP)
			_, err = makeGetRequest(url, "ws.mis.li", 404, true)
			Expect(err).To(BeNil())
		})

		It("[fifty-ingresses-with-services] should have get 200 for each ingress", func() {
			// create namespace
			namespaceName = "e2e-fifty-ingresses"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace: ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// create objects in the yaml
			path := "testdata/networking-v1/one-namespace-many-ingresses/fifty-ingresses-with-services/generated.yaml"
			klog.Info("Applying yaml: ", path)
			err = applyYaml(clientset, crdClient, namespaceName, path)
			Expect(err).To(BeNil())

			time.Sleep(30 * time.Second)
			// get ip address for 1 ingress
			klog.Info("Getting public IP from Ingress...")
			publicIP, err := getPublicIP(clientset, namespaceName)
			Expect(err).To(BeNil())
			Expect(publicIP).ToNot(Equal(""))

			// make curl request
			hosts := []string{"appa.", "appb."}
			url := fmt.Sprintf("https://%s/status/200", publicIP)
			klog.Infof("Sending get request %s ...", url)
			for i := 1; i <= 40; i++ {
				for _, host := range hosts {
					hostIndex := host + strconv.Itoa(i)
					klog.Infof("Sending request with host %s ...", hostIndex)
					_, err = makeGetRequest(url, hostIndex, 200, true)
					Expect(err).To(BeNil())
				}
			}
		})

		It("[hostname-with-wildcard] request host matchs hostname-extension annotation should work", func() {
			// create namespace
			namespaceName = "e2e-hostname-with-wildcard"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace: ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// create objects in the yaml
			path := "testdata/networking-v1/one-namespace-many-ingresses/hostname-with-wildcard/app.yaml"
			klog.Info("Applying yaml: ", path)
			err = applyYaml(clientset, crdClient, namespaceName, path)
			Expect(err).To(BeNil())

			time.Sleep(30 * time.Second)

			// get ip address for 1 ingress
			klog.Info("Getting public IP from Ingress...")
			publicIP, err := getPublicIP(clientset, namespaceName)
			Expect(err).To(BeNil())
			Expect(publicIP).ToNot(Equal(""))

			url := fmt.Sprintf("https://%s/status/200", publicIP)

			// simple hostname
			_, err = makeGetRequest(url, "www.extended.com", 200, true)
			Expect(err).To(BeNil())

			// wilcard host name on multiple hostnames wildcard listener
			_, err = makeGetRequest(url, "app.extended.com", 200, true)
			Expect(err).To(BeNil())

			// simple hostname with 1 host name which is wildcard hostname
			_, err = makeGetRequest(url, "www.singlequestionmarkhost.uk", 200, true)
			Expect(err).To(BeNil())

			// return 404 for random hostname
			_, err = makeGetRequest(url, "random.com", 404, true)
			Expect(err).To(BeNil())
		})

		It("[hostname-with-wildcard-priority] request host matchs hostname-extension annotation should work with priority", func() {
			// create namespace
			namespaceName = "e2e-hostname-with-wildcard-priority"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace: ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// create objects in the yaml
			path := "testdata/networking-v1/one-namespace-many-ingresses/hostname-with-wildcard-priority/app.yaml"
			klog.Info("Applying yaml: ", path)
			err = applyYaml(clientset, crdClient, namespaceName, path)
			Expect(err).To(BeNil())

			time.Sleep(30 * time.Second)

			// get ip address for 1 ingress
			klog.Info("Getting public IP from Ingress...")
			publicIP, err := getPublicIP(clientset, namespaceName)
			Expect(err).To(BeNil())
			Expect(publicIP).ToNot(Equal(""))

			// test url that only works with *.extended.com
			url := fmt.Sprintf("https://%s/anything", publicIP)

			// simple hostname
			_, err = makeGetRequest(url, "www.extended.com", 200, true)
			Expect(err).To(BeNil())

			// wilcard host name on multiple hostnames wildcard listener
			_, err = makeGetRequest(url, "app.extended.com", 200, true)
			Expect(err).To(BeNil())

			// return 404 for random hostname
			_, err = makeGetRequest(url, "random.com", 404, true)
			Expect(err).To(BeNil())

			// test url that only works with contoso.extended.com
			url = fmt.Sprintf("https://%s/status/200", publicIP)

			// simple hostname with 1 host name which is wildcard hostname with highest priority
			_, err = makeGetRequest(url, "contoso.extended.com", 200, true)
			Expect(err).To(BeNil())
		})

		It("[same-port-public-private] ingresses with same port on both public and private IP should work", func() {
			// create namespace
			namespaceName = "e2e-same-port-public-private"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace: ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// create objects in the yaml
			path := "testdata/networking-v1/one-namespace-many-ingresses/same-port-public-private/app.yaml"
			klog.Info("Applying yaml: ", path)
			err = applyYaml(clientset, crdClient, namespaceName, path)
			Expect(err).To(BeNil())

			var exampleComListeners []n.ApplicationGatewayHTTPListener
			// Check that gateway has two listeners eventually
			klog.Info("Checking that gateway has two listeners for hostname example.com...")
			Eventually(func() bool {
				appGW, err := getGateway()
				Expect(err).To(BeNil())

				bytes, _ := json.MarshalIndent(appGW.HTTPListeners, "", "  ")
				klog.Infof("Listeners: %s", bytes)

				exampleComListeners = []n.ApplicationGatewayHTTPListener{}
				for _, listener := range *appGW.HTTPListeners {
					if listener.HostNames == nil {
						continue
					}

					if len(*listener.HostNames) == 0 {
						continue
					}

					if (*listener.HostNames)[0] == "example.com" {
						exampleComListeners = append(exampleComListeners, listener)
					}
				}

				return len(exampleComListeners) == 2
			}, 10*time.Minute, 5*time.Second).Should(BeTrue())

			// Check that both listeners have the same frontend port
			klog.Info("Checking that both listeners have the same frontend port...")
			Expect(exampleComListeners[0].FrontendPort.ID).To(Equal(exampleComListeners[1].FrontendPort.ID))

			// Check that both listeners have the different frontend IP
			klog.Info("Checking that both listeners have the different frontend IP...")
			Expect(exampleComListeners[0].FrontendIPConfiguration.ID).ToNot(Equal(exampleComListeners[1].FrontendIPConfiguration.ID))
		})

		AfterEach(func() {
			// clear all namespaces
			cleanUp(clientset)
		})
	})
})
