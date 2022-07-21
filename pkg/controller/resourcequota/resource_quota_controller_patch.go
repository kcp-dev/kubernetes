/*
Copyright 2022 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resourcequota

import (
	"context"
	"fmt"
	"reflect"

	"github.com/kcp-dev/logicalcluster"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func (rq *Controller) KCPUpdateMonitors(ctx context.Context, clusterName logicalcluster.Name, discoveryFunc NamespacedResourcesFunc) {
	oldResources := make(map[schema.GroupVersionResource]struct{})
	// Get the current resource list from discovery.
	newResources, err := GetQuotableResources(discoveryFunc)
	if err != nil {
		utilruntime.HandleError(err)

		if discovery.IsGroupDiscoveryFailedError(err) && len(newResources) > 0 {
			// In partial discovery cases, don't remove any existing informers, just add new ones
			for k, v := range oldResources {
				newResources[k] = v
			}
		} else {
			// short circuit in non-discovery error cases or if discovery returned zero resources
			return
		}
	}

	// Decide whether discovery has reported a change.
	if reflect.DeepEqual(oldResources, newResources) {
		klog.V(4).Infof("%s: no resource updates from discovery, skipping resource quota sync", clusterName)
		return
	}

	// Ensure workers are paused to avoid processing events before informers
	// have resynced.
	rq.workerLock.Lock()
	defer rq.workerLock.Unlock()

	// Something has changed, so track the new state and perform a sync.
	if klog.V(2).Enabled() {
		klog.Infof("%s: syncing resource quota controller with updated resources from discovery: %s", clusterName, printDiff(oldResources, newResources))
	}

	// Perform the monitor resync and wait for controllers to report cache sync.
	if err := rq.resyncMonitors(newResources); err != nil {
		utilruntime.HandleError(fmt.Errorf("%s: failed to sync resource monitors: %v", clusterName, err))
		return
	}
	if rq.quotaMonitor != nil && !cache.WaitForNamedCacheSync(fmt.Sprintf("%q resource quota", clusterName), ctx.Done(), rq.quotaMonitor.IsSynced) {
		utilruntime.HandleError(fmt.Errorf("%s: timed out waiting for quota monitor sync", clusterName))
		return
	}

	// success, remember newly synced resources
	oldResources = newResources
	klog.V(2).Infof("%s: synced quota controller", clusterName)

	// List all the quotas (this is scoped to the workspace)
	quotas, err := rq.rqLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("%s: error listing all resourcequotas: %v", clusterName, err))
	}

	for i := range quotas {
		quota := quotas[i]
		klog.V(2).Infof("%s: enqueuing resourcequota %s/%s because the list of available APIs changed", clusterName, quota.Namespace, quota.Name)
		rq.addQuota(quota)
	}
}
