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

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	v1beta1 "k8s.io/client-go/kcp/listers/admissionregistration/v1beta1"
	versioned "k8s.io/client-go/kubernetes"

	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	admissionregistrationv1beta1listers "k8s.io/client-go/listers/admissionregistration/v1beta1"
)

// MutatingWebhookConfigurationInformer provides access to a shared informer and lister for
// MutatingWebhookConfigurations.
type MutatingWebhookConfigurationInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewMutatingWebhookConfigurationInformer constructs a new informer for MutatingWebhookConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewMutatingWebhookConfigurationInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredMutatingWebhookConfigurationInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredMutatingWebhookConfigurationInformer constructs a new informer for MutatingWebhookConfiguration type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredMutatingWebhookConfigurationInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return kcpinformers.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Watch(context.TODO(), options)
			},
		},
		&admissionregistrationv1beta1.MutatingWebhookConfiguration{},
		resyncPeriod,
		indexers,
	)
}

func (f MutatingWebhookConfigurationInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredMutatingWebhookConfigurationInformer(
		client,
		resyncPeriod,
		cache.Indexers{
			kcpcache.ClusterIndexName: kcpcache.ClusterIndexFunc,
		},
		f.tweakListOptions,
	)
}

func (f MutatingWebhookConfigurationInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&admissionregistrationv1beta1.MutatingWebhookConfiguration{}, f.defaultInformer)
}

func (f MutatingWebhookConfigurationInformer) Lister() admissionregistrationv1beta1listers.MutatingWebhookConfigurationLister {
	return v1beta1.NewMutatingWebhookConfigurationClusterLister(f.Informer().GetIndexer())
}
