// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

//go:build unittest
// +build unittest

package k8scontext

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	fakediscovery "k8s.io/client-go/discovery/fake"
	testclient "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	agrewritev1beta1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureapplicationgatewayrewrite/v1beta1"
	prohibitedv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	multiClusterIngress "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/multiclusteringress/v1alpha1"
	multiClusterService "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/multiclusterservice/v1alpha1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	agiccrdFake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/fake"
	mcsfake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned/fake"
	istiofake "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned/fake"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
)

var _ = ginkgo.Describe("Initial cache sync with CRD informers", func() {
	const (
		prohibitedResource = "azureingressprohibitedtargets"
		rewriteResource    = "azureapplicationgatewayrewrites"
		mciResource        = "multiclusteringresses"
		mcsResource        = "multiclusterservices"
	)
	prohibitedGVR := prohibitedv1.SchemeGroupVersion.WithResource(prohibitedResource)
	rewriteGVR := agrewritev1beta1.SchemeGroupVersion.WithResource(rewriteResource)
	mciGVR := multiClusterIngress.SchemeGroupVersion.WithResource(mciResource)
	mcsGVR := multiClusterService.SchemeGroupVersion.WithResource(mcsResource)

	var (
		k8sClient   *testclient.Clientset
		crdClient   *agiccrdFake.Clientset
		mcsClient   *mcsfake.Clientset
		ctxt        *Context
		stopChannel chan struct{}
		envVars     environment.EnvVariables
	)

	// serveCRDs makes API discovery report the given resources as served.
	serveCRDs := func(gvrs ...schema.GroupVersionResource) {
		discovery := k8sClient.Discovery().(*fakediscovery.FakeDiscovery)
		for _, gvr := range gvrs {
			discovery.Resources = append(discovery.Resources, &metav1.APIResourceList{
				GroupVersion: gvr.GroupVersion().String(),
				APIResources: []metav1.APIResource{{Name: gvr.Resource, Namespaced: true}},
			})
		}
	}

	// failDiscovery makes every API discovery request fail.
	failDiscovery := func() {
		k8sClient.PrependReactor("get", "resource", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewServiceUnavailable("discovery unavailable")
		})
	}

	// serveListFrom answers LIST requests for resource with list (after gate is closed, when gate is non-nil)
	// and WATCH requests with an idle watcher. The generated fake CRD clientsets cannot serve these themselves.
	serveListFrom := func(client *k8stesting.Fake, resource string, list runtime.Object, gate <-chan struct{}) {
		Expect(meta.NewAccessor().SetResourceVersion(list, "1")).To(Succeed())
		client.PrependReactor("list", resource, func(k8stesting.Action) (bool, runtime.Object, error) {
			if gate != nil {
				<-gate
			}
			return true, list, nil
		})
		client.PrependWatchReactor(resource, func(action k8stesting.Action) (bool, watch.Interface, error) {
			// The reflector first tries a streaming list (a WATCH without a resourceVersion); reject it like an
			// API server without WatchList support so that the reflector falls back to LIST + WATCH.
			if action.(k8stesting.WatchAction).GetWatchRestrictions().ResourceVersion == "" {
				return true, nil, apierrors.NewBadRequest("streaming lists are not supported")
			}
			return true, watch.NewFake(), nil
		})
	}

	serveList := func(resource string, list runtime.Object, gate <-chan struct{}) {
		serveListFrom(&crdClient.Fake, resource, list, gate)
	}

	protectManual := &prohibitedv1.AzureIngressProhibitedTargetList{
		Items: []prohibitedv1.AzureIngressProhibitedTarget{{
			ObjectMeta: metav1.ObjectMeta{Name: "protect-manual", Namespace: "test-ns"},
			Spec:       prohibitedv1.AzureIngressProhibitedTargetSpec{Hostname: "manual.example.com"},
		}},
	}

	expectProtectManualCached := func() {
		targets := ctxt.ListAzureProhibitedTargets()
		Expect(targets).To(HaveLen(1))
		Expect(targets[0].Spec.Hostname).To(Equal("manual.example.com"))
	}

	// failList answers LIST requests for resource the way an API server does when the CRD is not installed.
	failList := func(gvr schema.GroupVersionResource) {
		crdClient.PrependReactor("list", gvr.Resource, func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewNotFound(gvr.GroupResource(), "")
		})
	}

	runAsync := func() <-chan error {
		result := make(chan error, 1)
		go func() {
			result <- ctxt.Run(stopChannel, false, envVars)
		}()
		return result
	}

	ginkgo.BeforeEach(func() {
		stopChannel = make(chan struct{})
		k8sClient = testclient.NewSimpleClientset()
		crdClient = agiccrdFake.NewSimpleClientset()
		mcsClient = mcsfake.NewSimpleClientset()
		envVars = environment.GetFakeEnv()

		IsNetworkingV1PackageSupported = true
		ctxt = NewContext(k8sClient, crdClient, mcsClient, istiofake.NewSimpleClientset(), []string{"test-ns"}, 1000*time.Second, metricstore.NewFakeMetricStore(), envVars)
		Expect(ctxt).ToNot(BeNil())
	})

	ginkgo.AfterEach(func() {
		close(stopChannel)
	})

	ginkgo.It("returns only after AzureIngressProhibitedTargets are cached in shared mode", func() {
		envVars.EnableBrownfieldDeployment = true
		serveCRDs(prohibitedGVR, rewriteGVR)
		serveList(rewriteResource, &agrewritev1beta1.AzureApplicationGatewayRewriteList{}, nil)

		listAllowed := make(chan struct{})
		serveList(prohibitedResource, protectManual, listAllowed)

		runResult := runAsync()
		Consistently(runResult, 500*time.Millisecond).ShouldNot(Receive(), "Run must not return while the prohibited target LIST is in flight")

		close(listAllowed)
		Eventually(runResult, 5*time.Second).Should(Receive(BeNil()))
		Expect(ctxt.CacheSynced).To(BeClosed())
		expectProtectManualCached()
	})

	ginkgo.It("returns only after AzureIngressProhibitedTargets are cached in shared MultiCluster mode", func() {
		envVars.EnableBrownfieldDeployment = true
		envVars.MultiClusterMode = true
		serveCRDs(prohibitedGVR, mciGVR, mcsGVR)
		serveListFrom(&mcsClient.Fake, mciResource, &multiClusterIngress.MultiClusterIngressList{}, nil)
		serveListFrom(&mcsClient.Fake, mcsResource, &multiClusterService.MultiClusterServiceList{}, nil)

		listAllowed := make(chan struct{})
		serveList(prohibitedResource, protectManual, listAllowed)

		runResult := runAsync()
		Consistently(runResult, 500*time.Millisecond).ShouldNot(Receive(), "Run must not return while the prohibited target LIST is in flight")

		close(listAllowed)
		Eventually(runResult, 5*time.Second).Should(Receive(BeNil()))
		expectProtectManualCached()
	})

	ginkgo.It("does not block startup when the optional rewrite CRD is not installed", func() {
		failList(rewriteGVR)

		runResult := runAsync()
		Eventually(runResult, 5*time.Second).Should(Receive(BeNil()))
		Expect(ctxt.CacheSynced).To(BeClosed())
	})

	ginkgo.It("does not block shared mode when only the rewrite CRD is not installed", func() {
		envVars.EnableBrownfieldDeployment = true
		serveCRDs(prohibitedGVR)
		serveList(prohibitedResource, &prohibitedv1.AzureIngressProhibitedTargetList{}, nil)
		failList(rewriteGVR)

		runResult := runAsync()
		Eventually(runResult, 5*time.Second).Should(Receive(BeNil()))
	})

	ginkgo.It("fails closed in shared mode when the AzureIngressProhibitedTarget CRD is not installed", func() {
		envVars.EnableBrownfieldDeployment = true
		serveCRDs(rewriteGVR)
		serveList(rewriteResource, &agrewritev1beta1.AzureApplicationGatewayRewriteList{}, nil)
		failList(prohibitedGVR)

		runStop := make(chan struct{})
		runResult := make(chan error, 1)
		go func() {
			runResult <- ctxt.Run(runStop, false, envVars)
		}()

		Consistently(runResult, 500*time.Millisecond).ShouldNot(Receive())
		Expect(ctxt.CacheSynced).ToNot(BeClosed())

		close(runStop)
		var err error
		Eventually(runResult, 5*time.Second).Should(Receive(&err))
		var controllerErr *controllererrors.Error
		Expect(errors.As(err, &controllerErr)).To(BeTrue())
		Expect(controllerErr.Code).To(Equal(controllererrors.ErrorFailedInitialCacheSync))
	})

	ginkgo.It("waits for AzureIngressProhibitedTargets when API discovery fails", func() {
		envVars.EnableBrownfieldDeployment = true
		failDiscovery()
		failList(rewriteGVR)

		listAllowed := make(chan struct{})
		serveList(prohibitedResource, protectManual, listAllowed)

		runResult := runAsync()
		Consistently(runResult, 500*time.Millisecond).ShouldNot(Receive(), "Run must not return while the prohibited target LIST is in flight")

		close(listAllowed)
		Eventually(runResult, 5*time.Second).Should(Receive(BeNil()))
		expectProtectManualCached()
	})

	ginkgo.It("does not block startup when API discovery fails and the optional rewrite CRD is not installed", func() {
		failDiscovery()
		failList(rewriteGVR)

		runResult := runAsync()
		Eventually(runResult, 5*time.Second).Should(Receive(BeNil()))
	})
})

var _ = ginkgo.Describe("waitForNamedCacheSync", func() {
	const reportInterval = 10 * time.Millisecond

	var (
		mu      sync.Mutex
		reports [][]string
	)
	recordPending := func(pending []string) {
		mu.Lock()
		defer mu.Unlock()
		reports = append(reports, pending)
	}
	reportCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(reports)
	}
	lastReport := func() []string {
		mu.Lock()
		defer mu.Unlock()
		if len(reports) == 0 {
			return nil
		}
		return reports[len(reports)-1]
	}

	ginkgo.BeforeEach(func() {
		reports = nil
	})

	ginkgo.It("periodically reports the caches that are still syncing until all have synced", func() {
		var aSynced, bSynced atomic.Bool
		hasSynced := map[string]cache.InformerSynced{
			"b": bSynced.Load,
			"a": aSynced.Load,
			"c": func() bool { return true },
		}

		stopCh := make(chan struct{})
		defer close(stopCh)
		result := make(chan bool, 1)
		go func() {
			result <- waitForNamedCacheSync(stopCh, hasSynced, reportInterval, recordPending)
		}()

		Eventually(lastReport, time.Second).Should(Equal([]string{"a", "b"}))
		bSynced.Store(true)
		Eventually(lastReport, time.Second).Should(Equal([]string{"a"}))
		Expect(result).ToNot(Receive())

		aSynced.Store(true)
		Eventually(result, time.Second).Should(Receive(BeTrue()))
		reported := reportCount()
		Consistently(reportCount, 10*reportInterval).Should(Equal(reported), "no reports after returning")
	})

	ginkgo.It("returns false and stops reporting when stopped before the caches sync", func() {
		hasSynced := map[string]cache.InformerSynced{"a": func() bool { return false }}

		stopCh := make(chan struct{})
		result := make(chan bool, 1)
		go func() {
			result <- waitForNamedCacheSync(stopCh, hasSynced, reportInterval, recordPending)
		}()

		Eventually(reportCount, time.Second).Should(BeNumerically(">", 0))
		close(stopCh)
		Eventually(result, time.Second).Should(Receive(BeFalse()))
		reported := reportCount()
		Consistently(reportCount, 10*reportInterval).Should(Equal(reported), "no reports after returning")
	})
})
