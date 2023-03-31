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

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	upstreampolicyv1informers "k8s.io/client-go/informers/policy/v1"
	upstreampolicyv1listers "k8s.io/client-go/listers/policy/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/kcp-dev/client-go/informers/internalinterfaces"
	clientset "github.com/kcp-dev/client-go/kubernetes"
	policyv1listers "github.com/kcp-dev/client-go/listers/policy/v1"
)

// PodDisruptionBudgetClusterInformer provides access to a shared informer and lister for
// PodDisruptionBudgets.
type PodDisruptionBudgetClusterInformer interface {
	Cluster(logicalcluster.Name) upstreampolicyv1informers.PodDisruptionBudgetInformer
	Informer() kcpcache.ScopeableSharedIndexInformer
	Lister() policyv1listers.PodDisruptionBudgetClusterLister
}

type podDisruptionBudgetClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewPodDisruptionBudgetClusterInformer constructs a new informer for PodDisruptionBudget type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPodDisruptionBudgetClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredPodDisruptionBudgetClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredPodDisruptionBudgetClusterInformer constructs a new informer for PodDisruptionBudget type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPodDisruptionBudgetClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) kcpcache.ScopeableSharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PolicyV1().PodDisruptionBudgets().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.PolicyV1().PodDisruptionBudgets().Watch(context.TODO(), options)
			},
		},
		&policyv1.PodDisruptionBudget{},
		resyncPeriod,
		indexers,
	)
}

func (f *podDisruptionBudgetClusterInformer) defaultInformer(client clientset.ClusterInterface, resyncPeriod time.Duration) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredPodDisruptionBudgetClusterInformer(client, resyncPeriod, cache.Indexers{
		kcpcache.ClusterIndexName:             kcpcache.ClusterIndexFunc,
		kcpcache.ClusterAndNamespaceIndexName: kcpcache.ClusterAndNamespaceIndexFunc},
		f.tweakListOptions,
	)
}

func (f *podDisruptionBudgetClusterInformer) Informer() kcpcache.ScopeableSharedIndexInformer {
	return f.factory.InformerFor(&policyv1.PodDisruptionBudget{}, f.defaultInformer)
}

func (f *podDisruptionBudgetClusterInformer) Lister() policyv1listers.PodDisruptionBudgetClusterLister {
	return policyv1listers.NewPodDisruptionBudgetClusterLister(f.Informer().GetIndexer())
}

func (f *podDisruptionBudgetClusterInformer) Cluster(clusterName logicalcluster.Name) upstreampolicyv1informers.PodDisruptionBudgetInformer {
	return &podDisruptionBudgetInformer{
		informer: f.Informer().Cluster(clusterName),
		lister:   f.Lister().Cluster(clusterName),
	}
}

type podDisruptionBudgetInformer struct {
	informer cache.SharedIndexInformer
	lister   upstreampolicyv1listers.PodDisruptionBudgetLister
}

func (f *podDisruptionBudgetInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *podDisruptionBudgetInformer) Lister() upstreampolicyv1listers.PodDisruptionBudgetLister {
	return f.lister
}
