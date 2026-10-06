// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"sort"
	"strings"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// cacheSyncLogInterval is how often Run logs the informers whose initial sync is still pending.
const cacheSyncLogInterval = 30 * time.Second

// shouldWaitForCRDCache reports whether Run must wait for the initial sync of the CRD informer that serves gvr.
//
// The AzureIngressProhibitedTarget cache is always awaited: reconciling shared mode with an empty cache disables
// brownfield protection and lets AGIC take over the whole Application Gateway, so AGIC fails closed (and says why)
// when that CRD is not installed. Optional CRDs are awaited only when API discovery confirms they are served, so a
// missing CRD or a failed discovery request never blocks startup.
func (c *Context) shouldWaitForCRDCache(informer cache.SharedInformer, gvr schema.GroupVersionResource, name string) bool {
	served, err := c.isResourceServed(gvr)
	if informer == c.informers.AzureIngressProhibitedTarget {
		if err != nil {
			klog.Warningf("Could not determine whether %s is served by the API server; waiting for its cache to sync: %v", name, err)
		} else if !served {
			klog.Errorf("%s is not served by the API server; shared Application Gateway mode requires the AzureIngressProhibitedTarget CRD. AGIC will not reconcile until it is installed.", name)
		}
		return true
	}

	if err != nil {
		klog.Warningf("Could not determine whether %s is served by the API server; not waiting for its cache to sync: %v", name, err)
		return false
	}
	if !served {
		klog.Warningf("%s is not served by the API server (CRD not installed); not waiting for its cache to sync", name)
		return false
	}
	return true
}

// isResourceServed reports whether the API server serves gvr. It returns false only when the API
// server definitively reports the group/version or resource as absent (e.g. the CRD is not installed).
func (c *Context) isResourceServed(gvr schema.GroupVersionResource) (bool, error) {
	if c.kubeClient == nil {
		return true, nil
	}

	resources, err := c.kubeClient.Discovery().ServerResourcesForGroupVersion(gvr.GroupVersion().String())
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	for _, resource := range resources.APIResources {
		if resource.Name == gvr.Resource {
			return true, nil
		}
	}
	return false, nil
}

// waitForNamedCacheSync behaves like cache.WaitForCacheSync and, every reportInterval, passes the names of the
// caches that have not synced yet to reportPending. reportPending is never called after it returns.
func waitForNamedCacheSync(stopCh <-chan struct{}, hasSynced map[string]cache.InformerSynced, reportInterval time.Duration, reportPending func(pending []string)) bool {
	done := make(chan struct{})
	var reporter sync.WaitGroup
	reporter.Add(1)
	defer func() {
		close(done)
		reporter.Wait()
	}()

	go func() {
		defer reporter.Done()
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if pending := pendingCaches(hasSynced); len(pending) > 0 {
					reportPending(pending)
				}
			case <-done:
				return
			case <-stopCh:
				return
			}
		}
	}()

	syncFuncs := make([]cache.InformerSynced, 0, len(hasSynced))
	for _, name := range sortedKeys(hasSynced) {
		syncFuncs = append(syncFuncs, hasSynced[name])
	}
	return cache.WaitForCacheSync(stopCh, syncFuncs...)
}

func logPendingCaches(pending []string) {
	klog.Warningf("Still waiting for initial cache sync of: %s", strings.Join(pending, ", "))
}

func pendingCaches(hasSynced map[string]cache.InformerSynced) []string {
	var pending []string
	for _, name := range sortedKeys(hasSynced) {
		if !hasSynced[name]() {
			pending = append(pending, name)
		}
	}
	return pending
}

func sortedKeys(hasSynced map[string]cache.InformerSynced) []string {
	names := make([]string, 0, len(hasSynced))
	for name := range hasSynced {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
