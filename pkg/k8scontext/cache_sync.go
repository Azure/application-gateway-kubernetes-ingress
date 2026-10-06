// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package k8scontext

import (
	"sort"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// cacheSyncLogInterval is how often Run logs the informers whose initial sync is still pending.
const cacheSyncLogInterval = 30 * time.Second

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

// waitForNamedCacheSync behaves like cache.WaitForCacheSync and, every logInterval, logs which caches have not synced yet.
func waitForNamedCacheSync(stopCh <-chan struct{}, hasSynced map[string]cache.InformerSynced, logInterval time.Duration) bool {
	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(logInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if pending := pendingCaches(hasSynced); len(pending) > 0 {
					klog.Warningf("Still waiting for initial cache sync of: %s", strings.Join(pending, ", "))
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
