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

package v1beta2

import (
	"context"
	time "time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"

	kcpcache "github.com/kcp-dev/apimachinery/pkg/cache"
	kcpinformers "github.com/kcp-dev/apimachinery/third_party/informers"

	appsv1beta2 "k8s.io/api/apps/v1beta2"
	v1beta2 "k8s.io/client-go/kcp/listers/apps/v1beta2"
	versioned "k8s.io/client-go/kubernetes"

	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	appsv1beta2listers "k8s.io/client-go/listers/apps/v1beta2"
)

// StatefulSetInformer provides access to a shared informer and lister for
// StatefulSets.
type StatefulSetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewStatefulSetInformer constructs a new informer for StatefulSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewStatefulSetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredStatefulSetInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredStatefulSetInformer constructs a new informer for StatefulSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredStatefulSetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AppsV1beta2().StatefulSets(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AppsV1beta2().StatefulSets(namespace).Watch(context.TODO(), options)
			},
		},
		&appsv1beta2.StatefulSet{},
		resyncPeriod,
		indexers,
	)
}

func (f StatefulSetInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredStatefulSetInformer(
		client,
		f.namespace, resyncPeriod,
		cache.Indexers{
			kcpcache.ClusterIndexName:             kcpcache.ClusterIndexFunc,
			kcpcache.ClusterAndNamespaceIndexName: kcpcache.ClusterAndNamespaceIndexFunc},
		f.tweakListOptions,
	)
}

func (f StatefulSetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&appsv1beta2.StatefulSet{}, f.defaultInformer)
}

func (f StatefulSetInformer) Lister() appsv1beta2listers.StatefulSetLister {
	return v1beta2.NewStatefulSetClusterLister(f.Informer().GetIndexer())
}
