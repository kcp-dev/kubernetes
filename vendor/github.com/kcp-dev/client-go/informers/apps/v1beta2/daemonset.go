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

package v1beta2

import (
	"context"
	"time"

	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	kcpinformers "github.com/kcp-dev/apimachinery/v2/third_party/informers"
	"github.com/kcp-dev/logicalcluster/v3"

	appsv1beta2 "k8s.io/api/apps/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	upstreamappsv1beta2informers "k8s.io/client-go/informers/apps/v1beta2"
	upstreamappsv1beta2listers "k8s.io/client-go/listers/apps/v1beta2"
	"k8s.io/client-go/tools/cache"

	"github.com/kcp-dev/client-go/informers/internalinterfaces"
	clientset "github.com/kcp-dev/client-go/kubernetes"
	appsv1beta2listers "github.com/kcp-dev/client-go/listers/apps/v1beta2"
)

// DaemonSetClusterInformer provides access to a shared informer and lister for
// DaemonSets.
type DaemonSetClusterInformer interface {
	Cluster(logicalcluster.Name) upstreamappsv1beta2informers.DaemonSetInformer
	Informer() kcpcache.ScopeableSharedIndexInformer
	Lister() appsv1beta2listers.DaemonSetClusterLister
}

type daemonSetClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewDaemonSetClusterInformer constructs a new informer for DaemonSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewDaemonSetClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredDaemonSetClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredDaemonSetClusterInformer constructs a new informer for DaemonSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredDaemonSetClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) kcpcache.ScopeableSharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AppsV1beta2().DaemonSets().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AppsV1beta2().DaemonSets().Watch(context.TODO(), options)
			},
		},
		&appsv1beta2.DaemonSet{},
		resyncPeriod,
		indexers,
	)
}

func (f *daemonSetClusterInformer) defaultInformer(client clientset.ClusterInterface, resyncPeriod time.Duration) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredDaemonSetClusterInformer(client, resyncPeriod, cache.Indexers{
		kcpcache.ClusterIndexName:             kcpcache.ClusterIndexFunc,
		kcpcache.ClusterAndNamespaceIndexName: kcpcache.ClusterAndNamespaceIndexFunc},
		f.tweakListOptions,
	)
}

func (f *daemonSetClusterInformer) Informer() kcpcache.ScopeableSharedIndexInformer {
	return f.factory.InformerFor(&appsv1beta2.DaemonSet{}, f.defaultInformer)
}

func (f *daemonSetClusterInformer) Lister() appsv1beta2listers.DaemonSetClusterLister {
	return appsv1beta2listers.NewDaemonSetClusterLister(f.Informer().GetIndexer())
}

func (f *daemonSetClusterInformer) Cluster(clusterName logicalcluster.Name) upstreamappsv1beta2informers.DaemonSetInformer {
	return &daemonSetInformer{
		informer: f.Informer().Cluster(clusterName),
		lister:   f.Lister().Cluster(clusterName),
	}
}

type daemonSetInformer struct {
	informer cache.SharedIndexInformer
	lister   upstreamappsv1beta2listers.DaemonSetLister
}

func (f *daemonSetInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *daemonSetInformer) Lister() upstreamappsv1beta2listers.DaemonSetLister {
	return f.lister
}
