// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext_test

import (
	go_flag "flag"
	"reflect"
	"time"

	"github.com/getlantern/deepcopy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	istio_fake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

var _ = Describe("K8scontext", func() {
	var k8sClient kubernetes.Interface
	var ctxt *k8scontext.Context
	ingressNS := "test-ingress-controller"
	ingressName := "hello-world"
	var stopChannel chan struct{}

	// Create the "test-ingress-controller" namespace.
	// We will create all our resources under this namespace.
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ingressNS,
		},
	}

	// Create the Ingress resource.
	ingressObj := tests.NewIngressTestFixture(ingressNS, ingressName)
	ingress := &ingressObj

	// Create the Ingress resource.
	podObj := tests.NewPodTestFixture(ingressNS, "pod")
	pod := &podObj

	_ = go_flag.Lookup("logtostderr").Value.Set("true")
	_ = go_flag.Set("v", "5")

	// function to wait until sync
	waitContextSync := func(ctxt *k8scontext.Context, resourceList ...interface{}) {
		exists := make(map[interface{}]string)

		for {
			select {
			case in := <-ctxt.UpdateChannel.Out():
				event := in.(events.Event)
				for _, resource := range resourceList {
					if reflect.DeepEqual(resource, event.Value) {
						exists[resource] = ""
						break
					}
				}
			case <-time.After(1 * time.Second):
				break
			}

			if len(exists) == len(resourceList) {
				return
			}
		}
	}

	BeforeEach(func() {
		stopChannel = make(chan struct{})

		// Create the mock K8s client.
		k8sClient = testclient.NewSimpleClientset()
		crdClient := fake.NewSimpleClientset()
		istioCrdClient := istio_fake.NewSimpleClientset()

		_, err := k8sClient.CoreV1().Namespaces().Create(ns)
		Expect(err).Should(BeNil(), "Unable to create the namespace %s: %v", ingressNS, err)

		// create ingress in namespace
		_, err = k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Create(ingress)
		Expect(err).Should(BeNil(), "Unabled to create ingress resource due to: %v", err)

		// Create a `k8scontext` to start listening to ingress resources.
		ctxt = k8scontext.NewContext(k8sClient, crdClient, istioCrdClient, []string{ingressNS}, 1000*time.Second)

		Expect(ctxt).ShouldNot(BeNil(), "Unable to create `k8scontext`")
	})

	AfterEach(func() {
		close(stopChannel)
	})

	Context("Checking if we are able to listen to Ingress Resources", func() {
		It("Should be able to retrieve all Ingress Resources", func() {
			// Retrieve the Ingress to make sure it was created.
			ingresses, err := k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).List(metav1.ListOptions{})
			Expect(err).Should(BeNil(), "Unabled to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(1), "Expected to have a single ingress stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run(stopChannel, true, environment.GetFakeEnv())

			ingressListInterface := ctxt.Caches.Ingress.List()
			Expect(len(ingressListInterface)).To(Equal(1), "Expected to have a single ingress in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.ListHTTPIngresses()
			Expect(len(testIngresses)).To(Equal(1), "Expected to have a single ingress in the k8scontext but found: %d ingresses", len(testIngresses))

			// Make sure the ingress we got is the ingress we stored.
			Expect(testIngresses[0]).To(Equal(ingress), "Expected to retrieve the same ingress that we inserted, but it seems we found the following ingress: %v", testIngresses[0])

		})

		It("Should be able to follow modifications to the Ingress Resource.", func() {
			ingress.Spec.Rules[0].Host = "hellow-1.com"

			_, err := k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Update(ingress)
			Expect(err).Should(BeNil(), "Unabled to update ingress resource due to: %v", err)

			// Retrieve the Ingress to make sure it was updated.
			ingresses, err := k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).List(metav1.ListOptions{})
			Expect(err).Should(BeNil(), "Unable to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(1), "Expected to have a single ingress stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Due to the large sync time we don't expect the cache to be synced, till we force sync the cache.
			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run(stopChannel, true, environment.GetFakeEnv())

			ingressListInterface := ctxt.Caches.Ingress.List()
			// There should still be only one ingress resource.
			Expect(len(ingressListInterface)).To(Equal(1), "Expected to have a single ingress in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.ListHTTPIngresses()
			Expect(len(testIngresses)).To(Equal(1), "Expected to have a single ingress in the k8scontext but found: %d ingresses", len(testIngresses))
			// Make sure the ingress we got is the ingress we stored.
			Expect(testIngresses[0]).To(Equal(ingress), "Expected to retrieve the same ingress that we inserted, but it seems we found the following ingress: %v", testIngresses[0])

		})

		It("Should be able to follow deletion of the Ingress Resource.", func() {
			err := k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Delete(ingressName, nil)
			Expect(err).Should(BeNil(), "Unable to delete ingress resource due to: %v", err)

			// Retrieve the Ingress to make sure it was updated.
			ingresses, err := k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).List(metav1.ListOptions{})
			Expect(err).Should(BeNil(), "Unable to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(0), "Expected to have no ingresses stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Due to the large sync time we don't expect the cache to be synced, till we force sync the cache.
			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run(stopChannel, true, environment.GetFakeEnv())

			ingressListInterface := ctxt.Caches.Ingress.List()
			// There should still be only one ingress resource.
			Expect(len(ingressListInterface)).To(Equal(0), "Expected to have no ingress in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.ListHTTPIngresses()
			Expect(len(ingressListInterface)).To(Equal(0), "Expected to have no ingress in the k8scontext but found: %d ingresses", len(testIngresses))
		})

		It("Should be following Ingress Resource with Application Gateway specific annotations only.", func() {
			nonAppGWIngress := &v1beta1.Ingress{}
			deepcopy.Copy(nonAppGWIngress, ingress)
			nonAppGWIngress.Name = ingressName + "123"
			// Change the `Annotation` so that the controller doesn't see this Ingress.
			nonAppGWIngress.Annotations[annotations.IngressClassKey] = annotations.ApplicationGatewayIngressClass + "123"

			_, err := k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).Create(nonAppGWIngress)
			Expect(err).Should(BeNil(), "Unable to create non-Application Gateway ingress resource due to: %v", err)

			// Retrieve the Ingress to make sure it was updated.
			ingresses, err := k8sClient.ExtensionsV1beta1().Ingresses(ingressNS).List(metav1.ListOptions{})
			Expect(err).Should(BeNil(), "Unable to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(2), "Expected to have 2 ingresses stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Due to the large sync time we don't expect the cache to be synced, till we force sync the cache.
			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run(stopChannel, true, environment.GetFakeEnv())

			ingressListInterface := ctxt.Caches.Ingress.List()
			// There should two ingress resource.
			Expect(len(ingressListInterface)).To(Equal(2), "Expected to have 2 ingresses in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.ListHTTPIngresses()
			Expect(len(testIngresses)).To(Equal(1), "Expected to have a 1 ingress in the k8scontext but found: %d ingresses", len(testIngresses))

			// Make sure the ingress we got is the ingress we stored.
			Expect(testIngresses[0]).To(Equal(ingress), "Expected to retrieve the same ingress that we inserted, but it seems we found the following ingress: %v", testIngresses[0])
		})

		It("Should be able to follow add of the Pod Resource.", func() {
			_, err := k8sClient.CoreV1().Pods(ingressNS).Create(pod)
			Expect(err).Should(BeNil(), "Unable to create pod resource due to: %v", err)

			podObj1 := tests.NewPodTestFixture(ingressNS, "pod2")
			pod1 := &podObj1
			pod1.Labels = map[string]string{
				"app":   "pod2",
				"extra": "random",
			}
			_, err = k8sClient.CoreV1().Pods(ingressNS).Create(pod1)
			Expect(err).Should(BeNil(), "Unable to create pod resource due to: %v", err)

			// Retrieve the Pods to make sure it was updated.
			podList, err := k8sClient.CoreV1().Pods(ingressNS).List(metav1.ListOptions{})
			Expect(len(podList.Items)).To(Equal(2), "Expected to have two pod stored but found: %d pods", len(podList.Items))

			// Run context
			ctxt.Run(stopChannel, true, environment.GetFakeEnv())

			// Get and check that one of the pods exists.
			_, exists, _ := ctxt.Caches.Pods.Get(pod)
			Expect(exists).To(Equal(true), "Expected to find a pod in the cache: %d pods", len(podList.Items))

			// Search Pod list with a subset matching filter
			service := v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "service",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app": "pod2",
					},
				},
			}
			filteredPodList := ctxt.ListPodsByServiceSelector(service.Spec.Selector)
			Expect(len(filteredPodList)).To(Equal(1), "Expected to have filtered one pod with matching label: %d pods", len(podList.Items))

			// Search with a different filter
			service = v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "service",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app": "pod3",
					},
				},
			}
			filteredPodList = ctxt.ListPodsByServiceSelector(service.Spec.Selector)
			Expect(len(filteredPodList)).To(Equal(0), "Expected to find 0 pods with matching label: %d pods", len(podList.Items))
		})
	})

	Context("Checking if we are able to skip unrelated pod events", func() {
		It("should be able to select related pods", func() {
			// start context for syncing
			ctxt.Run(stopChannel, true, environment.GetFakeEnv())

			// create a POD with labels
			_, err := k8sClient.CoreV1().Pods(ingressNS).Create(pod)
			Expect(err).Should(BeNil(), "Unable to create pod resource due to: %v", err)

			// create a service with label
			servicePort := tests.NewServicePortsFixture()
			service := tests.NewServiceFixture(*servicePort...)
			service.Namespace = ingressNS
			_, err = k8sClient.CoreV1().Services(ingressNS).Create(service)
			Expect(err).Should(BeNil(), "Unable to create service resource due to: %v", err)

			// wait for sync
			waitContextSync(ctxt, ingress, pod, service)

			// check that ctxt synced the service
			Expect(len(ctxt.ListServices())).To(Equal(1), "Context was not able to sync in time")

			// run IsPodReferencedByAnyIngress: true
			Expect(ctxt.IsPodReferencedByAnyIngress(pod)).To(BeTrue(), "Expected is Pod is selected by the service and ingress.")
		})

		It("should be able to skip unrelated pods", func() {
			// start context for syncing
			ctxt.Run(stopChannel, true, environment.GetFakeEnv())

			// modify the labels on the POD
			pod.Labels = map[string]string{
				"random": "random",
			}
			_, err := k8sClient.CoreV1().Pods(ingressNS).Create(pod)
			Expect(err).Should(BeNil(), "Unable to create pod resource due to: %v", err)

			// create a service with label
			servicePort := tests.NewServicePortsFixture()
			service := tests.NewServiceFixture(*servicePort...)
			service.Namespace = ingressNS
			_, err = k8sClient.CoreV1().Services(ingressNS).Create(service)
			Expect(err).Should(BeNil(), "Unable to create service resource due to: %v", err)

			// wait for sync
			waitContextSync(ctxt, ingress, pod, service)

			// check that ctxt synced the service
			Expect(len(ctxt.ListServices())).To(Equal(1), "Context was not able to sync in time")

			// run IsPodReferencedByAnyIngress: false
			Expect(ctxt.IsPodReferencedByAnyIngress(pod)).To(BeFalse(), "Expected is Pod is not selected by the service and ingress.")
		})
	})

	Context("Checking IsEndpointReferencedByAnyIngress", func() {
		// create a endpoints with labels
		// create a service with label
		// create an ingress that uses that service
		// run IsPodReferencedByAnyIngress: true
		// ctxt.IsPodReferencedByAnyIngress()

		// modify ingress
		// run IsPodReferencedByAnyIngress: false
	})
})
