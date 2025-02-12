/*
Copyright The KCP Authors.

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

// Code generated by kcp code-generator. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	kcpinformers "github.com/kcp-dev/apimachinery/v2/third_party/informers"
	"github.com/kcp-dev/logicalcluster/v3"

	storagev1alpha1 "k8s.io/api/storage/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	upstreamstoragev1alpha1informers "k8s.io/client-go/informers/storage/v1alpha1"
	upstreamstoragev1alpha1listers "k8s.io/client-go/listers/storage/v1alpha1"
	"k8s.io/client-go/tools/cache"

	"github.com/kcp-dev/client-go/informers/internalinterfaces"
	clientset "github.com/kcp-dev/client-go/kubernetes"
	storagev1alpha1listers "github.com/kcp-dev/client-go/listers/storage/v1alpha1"
)

// CSIStorageCapacityClusterInformer provides access to a shared informer and lister for
// CSIStorageCapacities.
type CSIStorageCapacityClusterInformer interface {
	Cluster(logicalcluster.Name) upstreamstoragev1alpha1informers.CSIStorageCapacityInformer
	Informer() kcpcache.ScopeableSharedIndexInformer
	Lister() storagev1alpha1listers.CSIStorageCapacityClusterLister
}

type cSIStorageCapacityClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewCSIStorageCapacityClusterInformer constructs a new informer for CSIStorageCapacity type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCSIStorageCapacityClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredCSIStorageCapacityClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredCSIStorageCapacityClusterInformer constructs a new informer for CSIStorageCapacity type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCSIStorageCapacityClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) kcpcache.ScopeableSharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.StorageV1alpha1().CSIStorageCapacities().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.StorageV1alpha1().CSIStorageCapacities().Watch(context.TODO(), options)
			},
		},
		&storagev1alpha1.CSIStorageCapacity{},
		resyncPeriod,
		indexers,
	)
}

func (f *cSIStorageCapacityClusterInformer) defaultInformer(client clientset.ClusterInterface, resyncPeriod time.Duration) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredCSIStorageCapacityClusterInformer(client, resyncPeriod, cache.Indexers{
		kcpcache.ClusterIndexName:             kcpcache.ClusterIndexFunc,
		kcpcache.ClusterAndNamespaceIndexName: kcpcache.ClusterAndNamespaceIndexFunc},
		f.tweakListOptions,
	)
}

func (f *cSIStorageCapacityClusterInformer) Informer() kcpcache.ScopeableSharedIndexInformer {
	return f.factory.InformerFor(&storagev1alpha1.CSIStorageCapacity{}, f.defaultInformer)
}

func (f *cSIStorageCapacityClusterInformer) Lister() storagev1alpha1listers.CSIStorageCapacityClusterLister {
	return storagev1alpha1listers.NewCSIStorageCapacityClusterLister(f.Informer().GetIndexer())
}

func (f *cSIStorageCapacityClusterInformer) Cluster(clusterName logicalcluster.Name) upstreamstoragev1alpha1informers.CSIStorageCapacityInformer {
	return &cSIStorageCapacityInformer{
		informer: f.Informer().Cluster(clusterName),
		lister:   f.Lister().Cluster(clusterName),
	}
}

type cSIStorageCapacityInformer struct {
	informer cache.SharedIndexInformer
	lister   upstreamstoragev1alpha1listers.CSIStorageCapacityLister
}

func (f *cSIStorageCapacityInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *cSIStorageCapacityInformer) Lister() upstreamstoragev1alpha1listers.CSIStorageCapacityLister {
	return f.lister
}
