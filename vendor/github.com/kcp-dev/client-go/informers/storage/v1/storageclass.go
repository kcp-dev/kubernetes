//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

package v1

import (
	"context"
	"time"

	kcpcache "github.com/kcp-dev/apimachinery/pkg/cache"
	kcpinformers "github.com/kcp-dev/apimachinery/third_party/informers"
	"github.com/kcp-dev/logicalcluster/v2"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	upstreamstoragev1informers "k8s.io/client-go/informers/storage/v1"
	upstreamstoragev1listers "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kcp-dev/client-go/informers/internalinterfaces"
	clientset "github.com/kcp-dev/client-go/kubernetes"
	storagev1listers "github.com/kcp-dev/client-go/listers/storage/v1"
)

// StorageClassClusterInformer provides access to a shared informer and lister for
// StorageClasses.
type StorageClassClusterInformer interface {
	Cluster(logicalcluster.Name) upstreamstoragev1informers.StorageClassInformer
	Informer() kcpcache.ScopeableSharedIndexInformer
	Lister() storagev1listers.StorageClassClusterLister
}

type storageClassClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewStorageClassClusterInformer constructs a new informer for StorageClass type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewStorageClassClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredStorageClassClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredStorageClassClusterInformer constructs a new informer for StorageClass type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredStorageClassClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) kcpcache.ScopeableSharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.StorageV1().StorageClasses().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.StorageV1().StorageClasses().Watch(context.TODO(), options)
			},
		},
		&storagev1.StorageClass{},
		resyncPeriod,
		indexers,
	)
}

func (f *storageClassClusterInformer) defaultInformer(client clientset.ClusterInterface, resyncPeriod time.Duration) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredStorageClassClusterInformer(client, resyncPeriod, cache.Indexers{
		kcpcache.ClusterIndexName: kcpcache.ClusterIndexFunc,
	},
		f.tweakListOptions,
	)
}

func (f *storageClassClusterInformer) Informer() kcpcache.ScopeableSharedIndexInformer {
	return f.factory.InformerFor(&storagev1.StorageClass{}, f.defaultInformer)
}

func (f *storageClassClusterInformer) Lister() storagev1listers.StorageClassClusterLister {
	return storagev1listers.NewStorageClassClusterLister(f.Informer().GetIndexer())
}

func (f *storageClassClusterInformer) Cluster(cluster logicalcluster.Name) upstreamstoragev1informers.StorageClassInformer {
	return &storageClassInformer{
		informer: f.Informer().Cluster(cluster),
		lister:   f.Lister().Cluster(cluster),
	}
}

type storageClassInformer struct {
	informer cache.SharedIndexInformer
	lister   upstreamstoragev1listers.StorageClassLister
}

func (f *storageClassInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *storageClassInformer) Lister() upstreamstoragev1listers.StorageClassLister {
	return f.lister
}
