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

	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	kcpinformers "github.com/kcp-dev/apimachinery/v2/third_party/informers"
	"github.com/kcp-dev/logicalcluster/v3"

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

// CSINodeClusterInformer provides access to a shared informer and lister for
// CSINodes.
type CSINodeClusterInformer interface {
	Cluster(logicalcluster.Name) upstreamstoragev1informers.CSINodeInformer
	Informer() kcpcache.ScopeableSharedIndexInformer
	Lister() storagev1listers.CSINodeClusterLister
}

type cSINodeClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewCSINodeClusterInformer constructs a new informer for CSINode type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCSINodeClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredCSINodeClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredCSINodeClusterInformer constructs a new informer for CSINode type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCSINodeClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) kcpcache.ScopeableSharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.StorageV1().CSINodes().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.StorageV1().CSINodes().Watch(context.TODO(), options)
			},
		},
		&storagev1.CSINode{},
		resyncPeriod,
		indexers,
	)
}

func (f *cSINodeClusterInformer) defaultInformer(client clientset.ClusterInterface, resyncPeriod time.Duration) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredCSINodeClusterInformer(client, resyncPeriod, cache.Indexers{
		kcpcache.ClusterIndexName: kcpcache.ClusterIndexFunc,
	},
		f.tweakListOptions,
	)
}

func (f *cSINodeClusterInformer) Informer() kcpcache.ScopeableSharedIndexInformer {
	return f.factory.InformerFor(&storagev1.CSINode{}, f.defaultInformer)
}

func (f *cSINodeClusterInformer) Lister() storagev1listers.CSINodeClusterLister {
	return storagev1listers.NewCSINodeClusterLister(f.Informer().GetIndexer())
}

func (f *cSINodeClusterInformer) Cluster(clusterName logicalcluster.Name) upstreamstoragev1informers.CSINodeInformer {
	return &cSINodeInformer{
		informer: f.Informer().Cluster(clusterName),
		lister:   f.Lister().Cluster(clusterName),
	}
}

type cSINodeInformer struct {
	informer cache.SharedIndexInformer
	lister   upstreamstoragev1listers.CSINodeLister
}

func (f *cSINodeInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *cSINodeInformer) Lister() upstreamstoragev1listers.CSINodeLister {
	return f.lister
}
