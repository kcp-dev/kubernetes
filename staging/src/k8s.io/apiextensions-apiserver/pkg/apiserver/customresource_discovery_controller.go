/*
Copyright 2017 The Kubernetes Authors.

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

package apiserver

import (
	"fmt"
	"sort"
	"time"

	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/api/genericcontrolplanescheme"

	autoscaling "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	apiextensionshelpers "k8s.io/apiextensions-apiserver/pkg/apihelpers"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	informers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions/apiextensions/v1"
	listers "k8s.io/apiextensions-apiserver/pkg/client/listers/apiextensions/v1"
)

type DiscoveryController struct {
	versionHandler *versionDiscoveryHandler
	groupHandler   *groupDiscoveryHandler

	crdLister  listers.CustomResourceDefinitionLister
	crdsSynced cache.InformerSynced

	// To allow injection for testing.
	syncFn func(clusterGroupVersion discovery.ClusterGroupVersion) error

	queue workqueue.RateLimitingInterface
}

func NewDiscoveryController(crdInformer informers.CustomResourceDefinitionInformer, versionHandler *versionDiscoveryHandler, groupHandler *groupDiscoveryHandler) *DiscoveryController {
	c := &DiscoveryController{
		versionHandler: versionHandler,
		groupHandler:   groupHandler,
		crdLister:      crdInformer.Lister(),
		crdsSynced:     crdInformer.Informer().HasSynced,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DiscoveryController"),
	}

	crdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addCustomResourceDefinition,
		UpdateFunc: c.updateCustomResourceDefinition,
		DeleteFunc: c.deleteCustomResourceDefinition,
	})

	c.syncFn = c.sync

	return c
}

func (c *DiscoveryController) sync(clusterGroupVersion discovery.ClusterGroupVersion) error {

	apiVersionsForDiscovery := []metav1.GroupVersionForDiscovery{}
	apiResourcesForDiscovery := []metav1.APIResource{}
	versionsForDiscoveryMap := map[metav1.GroupVersion]bool{}

	crds, err := c.crdLister.List(labels.Everything())
	if err != nil {
		return err
	}
	foundVersion := false
	foundGroup := false
	for _, crd := range crds {
		if !apiextensionshelpers.IsCRDConditionTrue(crd, apiextensionsv1.Established) {
			continue
		}

		if crd.Spec.Group != clusterGroupVersion.Group {
			continue
		}

		if crd.GetClusterName() != clusterGroupVersion.ClusterName {
			continue
		}

		foundThisVersion := false
		var storageVersionHash string
		for _, v := range crd.Spec.Versions {
			if !v.Served {
				continue
			}
			// If there is any Served version, that means the group should show up in discovery
			foundGroup = true

			// HACK: support the case when we add core resources through CRDs (KCP scenario)
			groupVersion := crd.Spec.Group + "/" + v.Name
			if crd.Spec.Group == "" {
				groupVersion = v.Name
			}

			gv := metav1.GroupVersion{Group: crd.Spec.Group, Version: v.Name}

			if !versionsForDiscoveryMap[gv] {
				versionsForDiscoveryMap[gv] = true
				apiVersionsForDiscovery = append(apiVersionsForDiscovery, metav1.GroupVersionForDiscovery{
					GroupVersion: groupVersion,
					Version:      v.Name,
				})
			}
			if v.Name == clusterGroupVersion.Version {
				foundThisVersion = true
			}
			if v.Storage {
				storageVersionHash = discovery.StorageVersionHash(clusterGroupVersion.ClusterName, gv.Group, gv.Version, crd.Spec.Names.Kind)
			}
		}

		if !foundThisVersion {
			continue
		}
		foundVersion = true

		verbs := metav1.Verbs([]string{"delete", "deletecollection", "get", "list", "patch", "create", "update", "watch"})
		// if we're terminating we don't allow some verbs
		if apiextensionshelpers.IsCRDConditionTrue(crd, apiextensionsv1.Terminating) {
			verbs = metav1.Verbs([]string{"delete", "deletecollection", "get", "list", "watch"})
		}

		apiResourcesForDiscovery = append(apiResourcesForDiscovery, metav1.APIResource{
			Name:               crd.Status.AcceptedNames.Plural,
			SingularName:       crd.Status.AcceptedNames.Singular,
			Namespaced:         crd.Spec.Scope == apiextensionsv1.NamespaceScoped,
			Kind:               crd.Status.AcceptedNames.Kind,
			Verbs:              verbs,
			ShortNames:         crd.Status.AcceptedNames.ShortNames,
			Categories:         crd.Status.AcceptedNames.Categories,
			StorageVersionHash: storageVersionHash,
		})

		subresources, err := apiextensionshelpers.GetSubresourcesForVersion(crd, clusterGroupVersion.Version)
		if err != nil {
			return err
		}
		if subresources != nil && subresources.Status != nil {
			apiResourcesForDiscovery = append(apiResourcesForDiscovery, metav1.APIResource{
				Name:       crd.Status.AcceptedNames.Plural + "/status",
				Namespaced: crd.Spec.Scope == apiextensionsv1.NamespaceScoped,
				Kind:       crd.Status.AcceptedNames.Kind,
				Verbs:      metav1.Verbs([]string{"get", "patch", "update"}),
			})
		}

		if subresources != nil && subresources.Scale != nil {
			apiResourcesForDiscovery = append(apiResourcesForDiscovery, metav1.APIResource{
				Group:      autoscaling.GroupName,
				Version:    "v1",
				Kind:       "Scale",
				Name:       crd.Status.AcceptedNames.Plural + "/scale",
				Namespaced: crd.Spec.Scope == apiextensionsv1.NamespaceScoped,
				Verbs:      metav1.Verbs([]string{"get", "patch", "update"}),
			})
		}
	}

	sortGroupDiscoveryByKubeAwareVersion(apiVersionsForDiscovery)

	resourceListerFunc := discovery.APIResourceListerFunc(func() []metav1.APIResource {
		return apiResourcesForDiscovery
	})

	// HACK: if we are adding resources in legacy scheme group through CRDs (KCP scenario)
	// then do not expose the CRD `APIResource`s in their own CRD-related group`,
	// But instead add them in the existing legacy schema group
	if genericcontrolplanescheme.Scheme.IsGroupRegistered(clusterGroupVersion.Group) {
		if !foundGroup || !foundVersion {
			delete(discovery.ContributedResources, clusterGroupVersion)
		}

		discovery.ContributedResources[clusterGroupVersion] = resourceListerFunc
		return nil
	}

	if !foundGroup {
		c.groupHandler.unsetDiscovery(clusterGroupVersion.ClusterName, clusterGroupVersion.Group)
		c.versionHandler.unsetDiscovery(clusterGroupVersion.ClusterName, clusterGroupVersion.GroupVersion())
		return nil
	}

	if clusterGroupVersion.Group != "" {
		// If we don't add resources in the core API group
		apiGroup := metav1.APIGroup{
			Name:     clusterGroupVersion.Group,
			Versions: apiVersionsForDiscovery,
			// the preferred versions for a group is the first item in
			// apiVersionsForDiscovery after it put in the right ordered
			PreferredVersion: apiVersionsForDiscovery[0],
		}
		c.groupHandler.setDiscovery(clusterGroupVersion.ClusterName, clusterGroupVersion.Group, discovery.NewAPIGroupHandler(Codecs, apiGroup))

		if !foundVersion {
			c.versionHandler.unsetDiscovery(clusterGroupVersion.ClusterName, clusterGroupVersion.GroupVersion())
			return nil
		}
		c.versionHandler.setDiscovery(clusterGroupVersion.ClusterName, clusterGroupVersion.GroupVersion(), discovery.NewAPIVersionHandler(Codecs, clusterGroupVersion.GroupVersion(), resourceListerFunc))
	}

	return nil
}

func sortGroupDiscoveryByKubeAwareVersion(gd []metav1.GroupVersionForDiscovery) {
	sort.Slice(gd, func(i, j int) bool {
		return version.CompareKubeAwareVersionStrings(gd[i].Version, gd[j].Version) > 0
	})
}

func (c *DiscoveryController) Run(stopCh <-chan struct{}, synchedCh chan<- struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	defer klog.Info("Shutting down DiscoveryController")

	klog.Info("Starting DiscoveryController")

	if !cache.WaitForCacheSync(stopCh, c.crdsSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	// initially sync all group versions to make sure we serve complete discovery
	if err := wait.PollImmediateUntil(time.Second, func() (bool, error) {
		crds, err := c.crdLister.List(labels.Everything())
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to initially list CRDs: %v", err))
			return false, nil
		}
		for _, crd := range crds {
			for _, v := range crd.Spec.Versions {
				gv := discovery.ClusterGroupVersion{
					Group: crd.Spec.Group,
					Version: v.Name,
					ClusterName: crd.GetClusterName(),
				}
				if err := c.sync(gv); err != nil {
					utilruntime.HandleError(fmt.Errorf("failed to initially sync CRD version %v: %v", gv, err))
					return false, nil
				}
			}
		}
		return true, nil
	}, stopCh); err == wait.ErrWaitTimeout {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for discovery endpoint to initialize"))
		return
	} else if err != nil {
		panic(fmt.Errorf("unexpected error: %v", err))
	}
	close(synchedCh)

	// only start one worker thread since its a slow moving API
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *DiscoveryController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (c *DiscoveryController) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncFn(key.(discovery.ClusterGroupVersion))
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with: %v", key, err))
	c.queue.AddRateLimited(key)

	return true
}

func (c *DiscoveryController) enqueue(obj *apiextensionsv1.CustomResourceDefinition) {
	for _, v := range obj.Spec.Versions {
		c.queue.Add(discovery.ClusterGroupVersion{
			ClusterName: obj.GetClusterName(),
			Group:       obj.Spec.Group,
			Version:     v.Name})
	}
}

func (c *DiscoveryController) addCustomResourceDefinition(obj interface{}) {
	castObj := obj.(*apiextensionsv1.CustomResourceDefinition)
	klog.V(4).Infof("Adding customresourcedefinition %s", castObj.Name)
	c.enqueue(castObj)
}

func (c *DiscoveryController) updateCustomResourceDefinition(oldObj, newObj interface{}) {
	castNewObj := newObj.(*apiextensionsv1.CustomResourceDefinition)
	castOldObj := oldObj.(*apiextensionsv1.CustomResourceDefinition)
	klog.V(4).Infof("Updating customresourcedefinition %s", castOldObj.Name)
	// Enqueue both old and new object to make sure we remove and add appropriate Versions.
	// The working queue will resolve any duplicates and only changes will stay in the queue.
	c.enqueue(castNewObj)
	c.enqueue(castOldObj)
}

func (c *DiscoveryController) deleteCustomResourceDefinition(obj interface{}) {
	castObj, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		castObj, ok = tombstone.Obj.(*apiextensionsv1.CustomResourceDefinition)
		if !ok {
			klog.Errorf("Tombstone contained object that is not expected %#v", obj)
			return
		}
	}
	klog.V(4).Infof("Deleting customresourcedefinition %q", castObj.Name)
	c.enqueue(castObj)
}
