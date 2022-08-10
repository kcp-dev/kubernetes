//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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

package v1beta1

import (
	"context"
	time "time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"

	kcpcache "github.com/kcp-dev/apimachinery/pkg/cache"
	kcpinformers "github.com/kcp-dev/apimachinery/third_party/informers"

	coordinationv1beta1 "k8s.io/api/coordination/v1beta1"
	v1beta1 "k8s.io/client-go/kcp/listers/coordination/v1beta1"
	versioned "k8s.io/client-go/kubernetes"

	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	coordinationv1beta1listers "k8s.io/client-go/listers/coordination/v1beta1"
)

// LeaseInformer provides access to a shared informer and lister for
// Leases.
type LeaseInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewLeaseInformer constructs a new informer for Lease type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewLeaseInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredLeaseInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredLeaseInformer constructs a new informer for Lease type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredLeaseInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoordinationV1beta1().Leases(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoordinationV1beta1().Leases(namespace).Watch(context.TODO(), options)
			},
		},
		&coordinationv1beta1.Lease{},
		resyncPeriod,
		indexers,
	)
}

func (f LeaseInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredLeaseInformer(
		client,
		f.namespace, resyncPeriod,
		cache.Indexers{
			kcpcache.ClusterIndexName:             kcpcache.ClusterIndexFunc,
			kcpcache.ClusterAndNamespaceIndexName: kcpcache.ClusterAndNamespaceIndexFunc},
		f.tweakListOptions,
	)
}

func (f LeaseInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&coordinationv1beta1.Lease{}, f.defaultInformer)
}

func (f LeaseInformer) Lister() coordinationv1beta1listers.LeaseLister {
	return v1beta1.NewLeaseClusterLister(f.Informer().GetIndexer())
}
