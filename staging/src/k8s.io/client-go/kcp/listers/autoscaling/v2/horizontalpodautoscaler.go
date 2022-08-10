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

package v2

import (
	apimachinerycache "github.com/kcp-dev/apimachinery/pkg/cache"
	"github.com/kcp-dev/logicalcluster"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	autoscalingv2listers "k8s.io/client-go/listers/autoscaling/v2"
	"k8s.io/client-go/tools/cache"
)

var _ autoscalingv2listers.HorizontalPodAutoscalerLister = &HorizontalPodAutoscalerClusterLister{}

// HorizontalPodAutoscalerClusterLister implements the autoscalingv2listers.HorizontalPodAutoscalerLister interface.
type HorizontalPodAutoscalerClusterLister struct {
	indexer cache.Indexer
}

// NewHorizontalPodAutoscalerClusterLister returns a new HorizontalPodAutoscalerClusterLister.
func NewHorizontalPodAutoscalerClusterLister(indexer cache.Indexer) autoscalingv2listers.HorizontalPodAutoscalerLister {
	return &HorizontalPodAutoscalerClusterLister{indexer: indexer}
}

// List lists all autoscalingv2.HorizontalPodAutoscaler in the indexer.
func (s HorizontalPodAutoscalerClusterLister) List(selector labels.Selector) (ret []*autoscalingv2.HorizontalPodAutoscaler, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*autoscalingv2.HorizontalPodAutoscaler))
	})
	return ret, err
}

// HorizontalPodAutoscalers returns an object that can list and get autoscalingv2.HorizontalPodAutoscaler.
func (s HorizontalPodAutoscalerClusterLister) HorizontalPodAutoscalers(namespace string) autoscalingv2listers.HorizontalPodAutoscalerNamespaceLister {
	panic("Calling 'HorizontalPodAutoscalers' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get autoscalingv2.HorizontalPodAutoscaler.

func (s HorizontalPodAutoscalerClusterLister) Cluster(cluster logicalcluster.Name) autoscalingv2listers.HorizontalPodAutoscalerLister {
	return &HorizontalPodAutoscalerLister{indexer: s.indexer, cluster: cluster}
}

// HorizontalPodAutoscalerLister implements the autoscalingv2listers.HorizontalPodAutoscalerLister interface.
type HorizontalPodAutoscalerLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all autoscalingv2.HorizontalPodAutoscaler in the indexer.
func (s HorizontalPodAutoscalerLister) List(selector labels.Selector) (ret []*autoscalingv2.HorizontalPodAutoscaler, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*autoscalingv2.HorizontalPodAutoscaler)
		if selectAll {
			ret = append(ret, obj)
		} else {
			if selector.Matches(labels.Set(obj.GetLabels())) {
				ret = append(ret, obj)
			}
		}
	}

	return ret, err
}

// HorizontalPodAutoscalers returns an object that can list and get autoscalingv2.HorizontalPodAutoscaler.
func (s HorizontalPodAutoscalerLister) HorizontalPodAutoscalers(namespace string) autoscalingv2listers.HorizontalPodAutoscalerNamespaceLister {
	return &HorizontalPodAutoscalerNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// HorizontalPodAutoscalerNamespaceLister implements the autoscalingv2listers.HorizontalPodAutoscalerNamespaceLister interface.
type HorizontalPodAutoscalerNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all autoscalingv2.HorizontalPodAutoscaler in the indexer for a given namespace.
func (s HorizontalPodAutoscalerNamespaceLister) List(selector labels.Selector) (ret []*autoscalingv2.HorizontalPodAutoscaler, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*autoscalingv2.HorizontalPodAutoscaler)
		if selectAll {
			ret = append(ret, obj)
		} else {
			if selector.Matches(labels.Set(obj.GetLabels())) {
				ret = append(ret, obj)
			}
		}
	}
	return ret, err
}

// Get retrieves the autoscalingv2.HorizontalPodAutoscaler from the indexer for a given namespace and name.
func (s HorizontalPodAutoscalerNamespaceLister) Get(name string) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(autoscalingv2.Resource("HorizontalPodAutoscaler"), name)
	}
	return obj.(*autoscalingv2.HorizontalPodAutoscaler), nil
}
