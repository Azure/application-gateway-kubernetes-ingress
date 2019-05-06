// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext_test

import (
	go_flag "flag"
	"time"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/getlantern/deepcopy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("K8scontext", func() {
	var k8sClient kubernetes.Interface
	var ctxt *k8scontext.Context
	ingressNS := "test-ingress-controller"
	ingressName := "hello-world"

	// Create the "test-ingress-controller" namespace.
	// We will create all our resources under this namespace.
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ingressNS,
		},
	}

	// Create the Ingress resource.
	ingressObj := makeIngressTestFixture(ingressNS, ingressName)
	ingress := &ingressObj

	// Create the Ingress resource.
	podObj := makePodTestFixture(ingressNS, "pod")
	pod := &podObj

	go_flag.Lookup("logtostderr").Value.Set("true")
	go_flag.Set("v", "3")

	BeforeEach(func() {
		// Create the mock K8s client.
		k8sClient = testclient.NewSimpleClientset()

		_, err := k8sClient.CoreV1().Namespaces().Create(ns)
		Expect(err).Should(BeNil(), "Unable to create the namespace %s: %v", ingressNS, err)

		_, err = k8sClient.Extensions().Ingresses(ingressNS).Create(ingress)
		Expect(err).Should(BeNil(), "Unabled to create ingress resource due to: %v", err)

		// Create a `k8scontext` to start listiening to ingress resources.
		ctxt = k8scontext.NewContext(k8sClient, ingressNS, 1000*time.Second)
		Expect(ctxt).ShouldNot(BeNil(), "Unable to create `k8scontext`")
	})

	Context("Checking if we are able to listen to Ingress Resources", func() {
		It("Should be able to retrieve all Ingress Resources", func() {
			// Retrieve the Ingress to make sure it was created.
			ingresses, err := k8sClient.Extensions().Ingresses(ingressNS).List(metav1.ListOptions{})
			Expect(err).Should(BeNil(), "Unabled to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(1), "Expected to have a single ingress stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run()

			ingressListInterface := ctxt.Caches.Ingress.List()
			Expect(len(ingressListInterface)).To(Equal(1), "Expected to have a single ingress in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.GetHTTPIngressList()
			Expect(len(testIngresses)).To(Equal(1), "Expected to have a single ingress in the k8scontext but found: %d ingresses", len(testIngresses))

			// Make sure the ingress we got is the ingress we stored.
			Expect(testIngresses[0]).To(Equal(ingress), "Expected to retrieve the same ingress that we inserted, but it seems we found the following ingress: %v", testIngresses[0])

		})

		It("Should be able to follow modifications to the Ingress Resource.", func() {
			ingress.Spec.Rules[0].Host = "hellow-1.com"

			_, err := k8sClient.Extensions().Ingresses(ingressNS).Update(ingress)
			Expect(err).Should(BeNil(), "Unabled to update ingress resource due to: %v", err)

			// Retrieve the Ingress to make sure it was updated.
			ingresses, err := k8sClient.Extensions().Ingresses(ingressNS).List(metav1.ListOptions{})
			Expect(err).Should(BeNil(), "Unable to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(1), "Expected to have a single ingress stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Due to the large sync time we don't expect the cache to be synced, till we force sync the cache.
			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run()

			ingressListInterface := ctxt.Caches.Ingress.List()
			// There should still be only one ingress resource.
			Expect(len(ingressListInterface)).To(Equal(1), "Expected to have a single ingress in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.GetHTTPIngressList()
			Expect(len(testIngresses)).To(Equal(1), "Expected to have a single ingress in the k8scontext but found: %d ingresses", len(testIngresses))
			// Make sure the ingress we got is the ingress we stored.
			Expect(testIngresses[0]).To(Equal(ingress), "Expected to retrieve the same ingress that we inserted, but it seems we found the following ingress: %v", testIngresses[0])

		})

		It("Should be able to follow deletion of the Ingress Resource.", func() {
			err := k8sClient.Extensions().Ingresses(ingressNS).Delete(ingressName, nil)
			Expect(err).Should(BeNil(), "Unable to delete ingress resource due to: %v", err)

			// Retrieve the Ingress to make sure it was updated.
			ingresses, err := k8sClient.Extensions().Ingresses(ingressNS).List(metav1.ListOptions{})
			Expect(err).Should(BeNil(), "Unable to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(0), "Expected to have no ingresses stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Due to the large sync time we don't expect the cache to be synced, till we force sync the cache.
			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run()

			ingressListInterface := ctxt.Caches.Ingress.List()
			// There should still be only one ingress resource.
			Expect(len(ingressListInterface)).To(Equal(0), "Expected to have no ingress in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.GetHTTPIngressList()
			Expect(len(ingressListInterface)).To(Equal(0), "Expected to have no ingress in the k8scontext but found: %d ingresses", len(testIngresses))
		})

		It("Should be following Ingress Resource with Application Gateway specific annotations only.", func() {
			nonAppGWIngress := &v1beta1.Ingress{}
			deepcopy.Copy(nonAppGWIngress, ingress)
			nonAppGWIngress.Name = ingressName + "123"
			// Change the `Annotation` so that the controller doesn't see this Ingress.
			nonAppGWIngress.Annotations[annotations.IngressClassKey] = annotations.ApplicationGatewayIngressClass + "123"

			_, err := k8sClient.Extensions().Ingresses(ingressNS).Create(nonAppGWIngress)
			Expect(err).Should(BeNil(), "Unable to create non-Application Gateway ingress resource due to: %v", err)

			// Retrieve the Ingress to make sure it was updated.
			ingresses, err := k8sClient.Extensions().Ingresses(ingressNS).List(metav1.ListOptions{})
			Expect(err).Should(BeNil(), "Unable to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(2), "Expected to have 2 ingresses stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Due to the large sync time we don't expect the cache to be synced, till we force sync the cache.
			// Start the informers. This will sync the cache with the latest ingress.
			ctxt.Run()

			ingressListInterface := ctxt.Caches.Ingress.List()
			// There should two ingress resource.
			Expect(len(ingressListInterface)).To(Equal(2), "Expected to have 2 ingresses in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.GetHTTPIngressList()
			Expect(len(testIngresses)).To(Equal(1), "Expected to have a 1 ingress in the k8scontext but found: %d ingresses", len(testIngresses))

			// Make sure the ingress we got is the ingress we stored.
			Expect(testIngresses[0]).To(Equal(ingress), "Expected to retrieve the same ingress that we inserted, but it seems we found the following ingress: %v", testIngresses[0])
		})

		It("Should be able to follow add of the Pod Resource.", func() {
			_, err := k8sClient.CoreV1().Pods(ingressNS).Create(pod)
			Expect(err).Should(BeNil(), "Unable to create pod resource due to: %v", err)

			podObj1 := makePodTestFixture(ingressNS, "pod2")
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
			ctxt.Run()

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
			filteredPodList := ctxt.GetPodsByServiceSelector(service.Spec.Selector)
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
			filteredPodList = ctxt.GetPodsByServiceSelector(service.Spec.Selector)
			Expect(len(filteredPodList)).To(Equal(0), "Expected to find 0 pods with matching label: %d pods", len(podList.Items))
		})
	})
})

func makeIngressTestFixture(namespace string, ingressName string) v1beta1.Ingress {
	return v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
			Annotations: map[string]string{
				annotations.IngressClassKey: annotations.ApplicationGatewayIngressClass,
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				v1beta1.IngressRule{
					Host: "hello.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								v1beta1.HTTPIngressPath{
									Path: "/hi",
									Backend: v1beta1.IngressBackend{
										ServiceName: "hello-world",
										ServicePort: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func makePodTestFixture(namespace string, podName string) v1.Pod {
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "pod",
			},
		},
		Spec: v1.PodSpec{},
	}
}
