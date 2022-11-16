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

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	upstreambatchv1informers "k8s.io/client-go/informers/batch/v1"
	upstreambatchv1listers "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kcp-dev/client-go/informers/internalinterfaces"
	clientset "github.com/kcp-dev/client-go/kubernetes"
	batchv1listers "github.com/kcp-dev/client-go/listers/batch/v1"
)

// CronJobClusterInformer provides access to a shared informer and lister for
// CronJobs.
type CronJobClusterInformer interface {
	Cluster(logicalcluster.Name) upstreambatchv1informers.CronJobInformer
	Informer() kcpcache.ScopeableSharedIndexInformer
	Lister() batchv1listers.CronJobClusterLister
}

type cronJobClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewCronJobClusterInformer constructs a new informer for CronJob type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCronJobClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredCronJobClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredCronJobClusterInformer constructs a new informer for CronJob type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCronJobClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) kcpcache.ScopeableSharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BatchV1().CronJobs().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BatchV1().CronJobs().Watch(context.TODO(), options)
			},
		},
		&batchv1.CronJob{},
		resyncPeriod,
		indexers,
	)
}

func (f *cronJobClusterInformer) defaultInformer(client clientset.ClusterInterface, resyncPeriod time.Duration) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredCronJobClusterInformer(client, resyncPeriod, cache.Indexers{
		kcpcache.ClusterIndexName:             kcpcache.ClusterIndexFunc,
		kcpcache.ClusterAndNamespaceIndexName: kcpcache.ClusterAndNamespaceIndexFunc},
		f.tweakListOptions,
	)
}

func (f *cronJobClusterInformer) Informer() kcpcache.ScopeableSharedIndexInformer {
	return f.factory.InformerFor(&batchv1.CronJob{}, f.defaultInformer)
}

func (f *cronJobClusterInformer) Lister() batchv1listers.CronJobClusterLister {
	return batchv1listers.NewCronJobClusterLister(f.Informer().GetIndexer())
}

func (f *cronJobClusterInformer) Cluster(cluster logicalcluster.Name) upstreambatchv1informers.CronJobInformer {
	return &cronJobInformer{
		informer: f.Informer().Cluster(cluster),
		lister:   f.Lister().Cluster(cluster),
	}
}

type cronJobInformer struct {
	informer cache.SharedIndexInformer
	lister   upstreambatchv1listers.CronJobLister
}

func (f *cronJobInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *cronJobInformer) Lister() upstreambatchv1listers.CronJobLister {
	return f.lister
}
