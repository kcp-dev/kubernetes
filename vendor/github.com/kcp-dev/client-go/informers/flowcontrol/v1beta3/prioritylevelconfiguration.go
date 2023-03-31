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

package v1beta3

import (
	"context"
	"time"

	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	kcpinformers "github.com/kcp-dev/apimachinery/v2/third_party/informers"
	"github.com/kcp-dev/logicalcluster/v3"

	flowcontrolv1beta3 "k8s.io/api/flowcontrol/v1beta3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	upstreamflowcontrolv1beta3informers "k8s.io/client-go/informers/flowcontrol/v1beta3"
	upstreamflowcontrolv1beta3listers "k8s.io/client-go/listers/flowcontrol/v1beta3"
	"k8s.io/client-go/tools/cache"

	"github.com/kcp-dev/client-go/informers/internalinterfaces"
	clientset "github.com/kcp-dev/client-go/kubernetes"
	flowcontrolv1beta3listers "github.com/kcp-dev/client-go/listers/flowcontrol/v1beta3"
)

// PriorityLevelConfigurationClusterInformer provides access to a shared informer and lister for
// PriorityLevelConfigurations.
type PriorityLevelConfigurationClusterInformer interface {
	Cluster(logicalcluster.Name) upstreamflowcontrolv1beta3informers.PriorityLevelConfigurationInformer
	Informer() kcpcache.ScopeableSharedIndexInformer
	Lister() flowcontrolv1beta3listers.PriorityLevelConfigurationClusterLister
}

type priorityLevelConfigurationClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewPriorityLevelConfigurationClusterInformer constructs a new informer for PriorityLevelConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewPriorityLevelConfigurationClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredPriorityLevelConfigurationClusterInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredPriorityLevelConfigurationClusterInformer constructs a new informer for PriorityLevelConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredPriorityLevelConfigurationClusterInformer(client clientset.ClusterInterface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) kcpcache.ScopeableSharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.FlowcontrolV1beta3().PriorityLevelConfigurations().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.FlowcontrolV1beta3().PriorityLevelConfigurations().Watch(context.TODO(), options)
			},
		},
		&flowcontrolv1beta3.PriorityLevelConfiguration{},
		resyncPeriod,
		indexers,
	)
}

func (f *priorityLevelConfigurationClusterInformer) defaultInformer(client clientset.ClusterInterface, resyncPeriod time.Duration) kcpcache.ScopeableSharedIndexInformer {
	return NewFilteredPriorityLevelConfigurationClusterInformer(client, resyncPeriod, cache.Indexers{
		kcpcache.ClusterIndexName: kcpcache.ClusterIndexFunc,
	},
		f.tweakListOptions,
	)
}

func (f *priorityLevelConfigurationClusterInformer) Informer() kcpcache.ScopeableSharedIndexInformer {
	return f.factory.InformerFor(&flowcontrolv1beta3.PriorityLevelConfiguration{}, f.defaultInformer)
}

func (f *priorityLevelConfigurationClusterInformer) Lister() flowcontrolv1beta3listers.PriorityLevelConfigurationClusterLister {
	return flowcontrolv1beta3listers.NewPriorityLevelConfigurationClusterLister(f.Informer().GetIndexer())
}

func (f *priorityLevelConfigurationClusterInformer) Cluster(clusterName logicalcluster.Name) upstreamflowcontrolv1beta3informers.PriorityLevelConfigurationInformer {
	return &priorityLevelConfigurationInformer{
		informer: f.Informer().Cluster(clusterName),
		lister:   f.Lister().Cluster(clusterName),
	}
}

type priorityLevelConfigurationInformer struct {
	informer cache.SharedIndexInformer
	lister   upstreamflowcontrolv1beta3listers.PriorityLevelConfigurationLister
}

func (f *priorityLevelConfigurationInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

func (f *priorityLevelConfigurationInformer) Lister() upstreamflowcontrolv1beta3listers.PriorityLevelConfigurationLister {
	return f.lister
}
