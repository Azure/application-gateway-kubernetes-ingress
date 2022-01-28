// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"context"
	"reflect"
	"time"

	"github.com/getlantern/deepcopy"
	"github.com/knative/pkg/apis/istio/v1alpha3"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	multiClusterFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned/fake"
	istioFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

var _ = ginkgo.Describe("K8scontext", func() {
	var k8sClient kubernetes.Interface
	var ctxt *Context
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

	// function to wait until sync
	waitContextSync := func(ctxt *Context, resourceList ...interface{}) {
		exists := make(map[interface{}]string)

		for {
			select {
			case event := <-ctxt.Work:
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

	ginkgo.BeforeEach(func() {
		stopChannel = make(chan struct{})

		// Create the mock K8s client.
		k8sClient = testclient.NewSimpleClientset()
		crdClient := fake.NewSimpleClientset()
		istioCrdClient := istioFake.NewSimpleClientset()
		multiClusterCrdClient := multiClusterFake.NewSimpleClientset()

		_, err := k8sClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred(), "Unable to create the namespace %s: %v", ingressNS, err)

		// create ingress in namespace
		_, err = k8sClient.NetworkingV1().Ingresses(ingressNS).Create(context.TODO(), ingress, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred(), "Unabled to create ingress resource due to: %v", err)

		// Create a `k8scontext` to start listening to ingress resources.
		IsNetworkingV1PackageSupported = true
		ctxt = NewContext(k8sClient, crdClient, multiClusterCrdClient, istioCrdClient, []string{ingressNS}, 1000*time.Second, metricstore.NewFakeMetricStore(), environment.GetFakeEnv())

		Expect(ctxt).ShouldNot(BeNil(), "Unable to create `k8scontext`")
	})

	ginkgo.AfterEach(func() {
		close(stopChannel)
	})

	ginkgo.Context("Checking if we are able to listen to Ingress Resources", func() {
		ginkgo.It("Should be able to retrieve all Ingress Resources", func() {
			// Retrieve the Ingress to make sure it was created.
			ingresses, err := k8sClient.NetworkingV1().Ingresses(ingressNS).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unabled to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(1), "Expected to have a single ingress stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Start the informers. This will sync the cache with the latest ingress.
			runErr := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(runErr).ToNot(HaveOccurred())

			ingressListInterface := ctxt.Caches.Ingress.List()
			Expect(len(ingressListInterface)).To(Equal(1), "Expected to have a single ingress in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.ListHTTPIngresses()
			Expect(len(testIngresses)).To(Equal(1), "Expected to have a single ingress in the k8scontext but found: %d ingresses", len(testIngresses))

			// Make sure the ingress we got is the ingress we stored.
			Expect(testIngresses[0]).To(Equal(ingress), "Expected to retrieve the same ingress that we inserted, but it seems we found the following ingress: %v", testIngresses[0])

		})

		ginkgo.It("Should be able to follow modifications to the Ingress Resource.", func() {
			ingress.Spec.Rules[0].Host = "hellow-1.com"

			_, err := k8sClient.NetworkingV1().Ingresses(ingressNS).Update(context.TODO(), ingress, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unabled to update ingress resource due to: %v", err)

			// Retrieve the Ingress to make sure it was updated.
			ingresses, err := k8sClient.NetworkingV1().Ingresses(ingressNS).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(1), "Expected to have a single ingress stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Due to the large sync time we don't expect the cache to be synced, till we force sync the cache.
			// Start the informers. This will sync the cache with the latest ingress.
			runErr := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(runErr).ToNot(HaveOccurred())

			ingressListInterface := ctxt.Caches.Ingress.List()
			// There should still be only one ingress resource.
			Expect(len(ingressListInterface)).To(Equal(1), "Expected to have a single ingress in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.ListHTTPIngresses()
			Expect(len(testIngresses)).To(Equal(1), "Expected to have a single ingress in the k8scontext but found: %d ingresses", len(testIngresses))
			// Make sure the ingress we got is the ingress we stored.
			Expect(testIngresses[0]).To(Equal(ingress), "Expected to retrieve the same ingress that we inserted, but it seems we found the following ingress: %v", testIngresses[0])

		})

		ginkgo.It("Should be able to follow deletion of the Ingress Resource.", func() {
			err := k8sClient.NetworkingV1().Ingresses(ingressNS).Delete(context.TODO(), ingressName, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to delete ingress resource due to: %v", err)

			// Retrieve the Ingress to make sure it was updated.
			ingresses, err := k8sClient.NetworkingV1().Ingresses(ingressNS).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(0), "Expected to have no ingresses stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Due to the large sync time we don't expect the cache to be synced, till we force sync the cache.
			// Start the informers. This will sync the cache with the latest ingress.
			runErr := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(runErr).ToNot(HaveOccurred())

			ingressListInterface := ctxt.Caches.Ingress.List()
			// There should still be only one ingress resource.
			Expect(len(ingressListInterface)).To(Equal(0), "Expected to have no ingress in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.ListHTTPIngresses()
			Expect(len(ingressListInterface)).To(Equal(0), "Expected to have no ingress in the k8scontext but found: %d ingresses", len(testIngresses))
		})

		ginkgo.It("Should be following Ingress Resource with Application Gateway specific annotations only.", func() {
			nonAppGWIngress := &networking.Ingress{}
			err := deepcopy.Copy(nonAppGWIngress, ingress)
			Expect(err).ToNot(HaveOccurred())

			nonAppGWIngress.Name = ingressName + "123"
			// Change the `Annotation` so that the controller doesn't see this Ingress.
			nonAppGWIngress.Annotations[annotations.IngressClassKey] = environment.DefaultIngressClassController + "123"

			_, err = k8sClient.NetworkingV1().Ingresses(ingressNS).Create(context.TODO(), nonAppGWIngress, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create non-Application Gateway ingress resource due to: %v", err)

			// Retrieve the Ingress to make sure it was updated.
			ingresses, err := k8sClient.NetworkingV1().Ingresses(ingressNS).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to retrieve stored ingresses resource due to: %v", err)
			Expect(len(ingresses.Items)).To(Equal(2), "Expected to have 2 ingresses stored in mock K8s but found: %d ingresses", len(ingresses.Items))

			// Due to the large sync time we don't expect the cache to be synced, till we force sync the cache.
			// Start the informers. This will sync the cache with the latest ingress.
			runErr := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(runErr).ToNot(HaveOccurred())

			ingressListInterface := ctxt.Caches.Ingress.List()
			// There should two ingress resource.
			Expect(len(ingressListInterface)).To(Equal(2), "Expected to have 2 ingresses in the cache but found: %d ingresses", len(ingressListInterface))

			// Retrive the ingresses learnt by the controller.
			testIngresses := ctxt.ListHTTPIngresses()
			Expect(len(testIngresses)).To(Equal(1), "Expected to have a 1 ingress in the k8scontext but found: %d ingresses", len(testIngresses))

			// Make sure the ingress we got is the ingress we stored.
			Expect(testIngresses[0]).To(Equal(ingress), "Expected to retrieve the same ingress that we inserted, but it seems we found the following ingress: %v", testIngresses[0])
		})

		ginkgo.It("Should be able to follow add of the Pod Resource.", func() {
			_, err := k8sClient.CoreV1().Pods(ingressNS).Create(context.TODO(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create pod resource due to: %v", err)

			podObj1 := tests.NewPodTestFixture(ingressNS, "pod2")
			pod1 := &podObj1
			pod1.Namespace = "test-ingress-controller"
			pod1.Labels = map[string]string{
				"app":   "pod2",
				"extra": "random",
			}
			_, err = k8sClient.CoreV1().Pods(ingressNS).Create(context.TODO(), pod1, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create pod resource due to: %v", err)

			// Retrieve the Pods to make sure it was updated.
			podList, err := k8sClient.CoreV1().Pods(ingressNS).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(podList).ToNot(BeNil())
			Expect(len(podList.Items)).To(Equal(2), "Expected to have two pod stored but found: %d pods", len(podList.Items))

			// Run context
			runErr := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(runErr).ToNot(HaveOccurred())

			// Get and check that one of the pods exists.
			_, exists, _ := ctxt.Caches.Pods.Get(pod)
			Expect(exists).To(Equal(true), "Expected to find a pod in the cache: %d pods", len(podList.Items))

			// Search Pod list with a subset matching filter
			service := v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "test-ingress-controller",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app": "pod2",
					},
				},
			}
			filteredPodList := ctxt.ListPodsByServiceSelector(&service)
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
			filteredPodList = ctxt.ListPodsByServiceSelector(&service)
			Expect(len(filteredPodList)).To(Equal(0), "Expected to find 0 pods with matching label: %d pods", len(podList.Items))

			// Filter with a same pod label but different namespace
			service = v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "different-namespace",
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app": "pod2",
					},
				},
			}
			filteredPodList = ctxt.ListPodsByServiceSelector(&service)
			Expect(len(filteredPodList)).To(Equal(0), "Expected to find 0 pods with matching label but found: %d pods", len(filteredPodList))
		})
	})

	ginkgo.Context("Checking if we are able to skip unrelated pod events", func() {
		ginkgo.It("should be able to select related pods", func() {
			// start context for syncing
			runErr := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(runErr).ToNot(HaveOccurred())

			// create a POD with labels
			_, err := k8sClient.CoreV1().Pods(ingressNS).Create(context.TODO(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create pod resource due to: %v", err)

			// create a service with label
			servicePort := tests.NewServicePortsFixture()
			service := tests.NewServiceFixture(*servicePort...)
			service.Namespace = ingressNS
			_, err = k8sClient.CoreV1().Services(ingressNS).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create service resource due to: %v", err)

			// wait for sync
			waitContextSync(ctxt, ingress, pod, service)

			// check that ctxt synced the service
			Expect(len(ctxt.ListServices())).To(Equal(1), "Context was not able to sync in time")

			// run IsPodReferencedByAnyIngress: true
			Expect(ctxt.IsPodReferencedByAnyIngress(pod)).To(BeTrue(), "Expected is Pod is selected by the service and ingress.")
		})

		ginkgo.It("should be able to skip unrelated pods", func() {
			// start context for syncing
			runErr := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(runErr).ToNot(HaveOccurred())

			// modify the labels on the POD
			pod.Labels = map[string]string{
				"random": "random",
			}
			_, err := k8sClient.CoreV1().Pods(ingressNS).Create(context.TODO(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create pod resource due to: %v", err)

			// create a service with label
			servicePort := tests.NewServicePortsFixture()
			service := tests.NewServiceFixture(*servicePort...)
			service.Namespace = ingressNS
			_, err = k8sClient.CoreV1().Services(ingressNS).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create service resource due to: %v", err)

			// wait for sync
			waitContextSync(ctxt, ingress, pod, service)

			// check that ctxt synced the service
			Expect(len(ctxt.ListServices())).To(Equal(1), "Context was not able to sync in time")

			// run IsPodReferencedByAnyIngress: false
			Expect(ctxt.IsPodReferencedByAnyIngress(pod)).To(BeFalse(), "Expected is Pod is not selected by the service and ingress.")
		})
	})

	ginkgo.Context("Checking if we are able to skip unrelated endpoints events", func() {
		ginkgo.It("should be able to select related endpoints", func() {
			// start context for syncing
			runErr := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(runErr).ToNot(HaveOccurred())

			endpoints := tests.NewEndpointsFixture()
			endpoints.Namespace = ingressNS

			// create a POD with labels
			_, err := k8sClient.CoreV1().Endpoints(ingressNS).Create(context.TODO(), endpoints, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create endpoints resource due to: %v", err)

			// create a service with label
			servicePort := tests.NewServicePortsFixture()
			service := tests.NewServiceFixture(*servicePort...)
			service.Namespace = ingressNS
			_, err = k8sClient.CoreV1().Services(ingressNS).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create service resource due to: %v", err)

			// wait for sync
			waitContextSync(ctxt, ingress, service, endpoints)

			// check that ctxt synced the service
			Expect(len(ctxt.ListServices())).To(Equal(1), "Context was not able to sync in time")

			// run IsPodReferencedByAnyIngress: true
			Expect(ctxt.IsEndpointReferencedByAnyIngress(endpoints)).To(BeTrue(), "Expected is endpoints is selected by the service and ingress.")
		})

		ginkgo.It("should be able to skip unrelated endpoints", func() {
			// start context for syncing
			runErr := ctxt.Run(stopChannel, true, environment.GetFakeEnv())
			Expect(runErr).ToNot(HaveOccurred())

			endpoints := tests.NewEndpointsFixture()
			endpoints.Name = "random"
			endpoints.Namespace = ingressNS

			// create a POD with labels
			_, err := k8sClient.CoreV1().Endpoints(ingressNS).Create(context.TODO(), endpoints, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create endpoints resource due to: %v", err)

			// create a service with label
			servicePort := tests.NewServicePortsFixture()
			service := tests.NewServiceFixture(*servicePort...)
			service.Namespace = ingressNS
			_, err = k8sClient.CoreV1().Services(ingressNS).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Unable to create service resource due to: %v", err)

			// wait for sync
			waitContextSync(ctxt, ingress, service, endpoints)

			// check that ctxt synced the service
			Expect(len(ctxt.ListServices())).To(Equal(1), "Context was not able to sync in time")

			// run IsPodReferencedByAnyIngress: true
			Expect(ctxt.IsEndpointReferencedByAnyIngress(endpoints)).To(BeFalse(), "Expected is endpoints is not selected by the service and ingress.")
		})
	})

	ginkgo.Context("Checking AddIngressStatus and RemoveIngressStatus", func() {
		ip := IPAddress("address")
		ginkgo.It("adds IP when not present and then removes", func() {
			// add test
			err := ctxt.UpdateIngressStatus(*ingress, ip)
			Expect(err).ToNot(HaveOccurred())
			updatedIngress, _ := k8sClient.NetworkingV1().Ingresses(ingress.Namespace).Get(context.TODO(), ingress.Name, metav1.GetOptions{})
			Expect(updatedIngress.Status.LoadBalancer.Ingress).Should(ContainElement(v1.LoadBalancerIngress{
				Hostname: "",
				IP:       string(ip),
			}))
			Expect(len(updatedIngress.Status.LoadBalancer.Ingress)).To(Equal(1))
		})

		ginkgo.It("doesn't add IP again when already present", func() {
			// add
			err := ctxt.UpdateIngressStatus(*ingress, ip)
			Expect(err).ToNot(HaveOccurred())
			// add again
			err = ctxt.UpdateIngressStatus(*ingress, ip)
			Expect(err).ToNot(HaveOccurred())
			updatedIngress, _ := k8sClient.NetworkingV1().Ingresses(ingress.Namespace).Get(context.TODO(), ingress.Name, metav1.GetOptions{})
			Expect(updatedIngress.Status.LoadBalancer.Ingress).Should(ContainElement(v1.LoadBalancerIngress{
				Hostname: "",
				IP:       string(ip),
			}))
			Expect(len(updatedIngress.Status.LoadBalancer.Ingress)).To(Equal(1))
		})
	})

	ginkgo.Context("Filtering Ingress Resources", func() {
		ginkgo.It("keep ingress, which does not have rules, but has a default backend", func() {
			ingr := tests.GetVerySimpleIngress()
			ingrList := []*networking.Ingress{
				ingr,
			}
			finalList := ctxt.filterAndSort(ingrList)
			Expect(finalList).To(ContainElement(ingr))
		})
	})

	ginkgo.Context("Check IsIstioGatewayIngress", func() {
		ginkgo.BeforeEach(func() {
			ctxt.ingressClassControllerName = environment.DefaultIngressClassController
		})

		ginkgo.It("returns error when gateway has no annotations", func() {
			gateway := &v1alpha3.Gateway{}
			actual := ctxt.IsIstioGatewayIngress(gateway)
			Expect(actual).To(Equal(false))
		})

		ginkgo.It("returns true with correct annotation", func() {
			gateway := &v1alpha3.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotations.IstioGatewayKey: environment.DefaultIngressClassController,
					},
				},
			}
			actual := ctxt.IsIstioGatewayIngress(gateway)
			Expect(actual).To(Equal(true))
		})
	})

	ginkgo.Context("Check IsApplicationGatewayIngress", func() {
		ginkgo.BeforeEach(func() {
			ctxt.ingressClassControllerName = environment.DefaultIngressClassController
		})

		ginkgo.It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual := ctxt.IsIngressClass(ing)
			Expect(actual).To(Equal(false))
		})

		ginkgo.It("returns true with correct annotation", func() {
			actual := ctxt.IsIngressClass(ingress)
			Expect(actual).To(Equal(true))
		})

		ginkgo.It("returns true with correct annotation", func() {
			ingress.Annotations[annotations.IngressClassKey] = "custom-class"
			ctxt.ingressClassControllerName = "custom-class"
			actual := ctxt.IsIngressClass(ingress)
			Expect(actual).To(Equal(true))
		})

		ginkgo.It("returns false with incorrect annotation", func() {
			ingress.Annotations[annotations.IngressClassKey] = "custom-class"
			actual := ctxt.IsIngressClass(ingress)
			Expect(ctxt.ingressClassControllerName).To(Equal(environment.DefaultIngressClassController))
			Expect(actual).To(Equal(false))
		})
	})

})
