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

package v1

import (
	apimachinerycache "github.com/kcp-dev/apimachinery/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v2"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
)

var _ appsv1listers.StatefulSetLister = &StatefulSetClusterLister{}

// StatefulSetClusterLister implements the appsv1listers.StatefulSetLister interface.
type StatefulSetClusterLister struct {
	indexer cache.Indexer
}

// NewStatefulSetClusterLister returns a new StatefulSetClusterLister.
func NewStatefulSetClusterLister(indexer cache.Indexer) appsv1listers.StatefulSetLister {
	return &StatefulSetClusterLister{indexer: indexer}
}

// List lists all appsv1.StatefulSet in the indexer.
func (s StatefulSetClusterLister) List(selector labels.Selector) (ret []*appsv1.StatefulSet, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*appsv1.StatefulSet))
	})
	return ret, err
}

// StatefulSets returns an object that can list and get appsv1.StatefulSet.
func (s StatefulSetClusterLister) StatefulSets(namespace string) appsv1listers.StatefulSetNamespaceLister {
	panic("Calling 'StatefulSets' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get appsv1.StatefulSet.

func (s StatefulSetClusterLister) Cluster(cluster logicalcluster.Name) appsv1listers.StatefulSetLister {
	return &StatefulSetLister{indexer: s.indexer, cluster: cluster}
}

// StatefulSetLister implements the appsv1listers.StatefulSetLister interface.
type StatefulSetLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all appsv1.StatefulSet in the indexer.
func (s StatefulSetLister) List(selector labels.Selector) (ret []*appsv1.StatefulSet, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*appsv1.StatefulSet)
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

// StatefulSets returns an object that can list and get appsv1.StatefulSet.
func (s StatefulSetLister) StatefulSets(namespace string) appsv1listers.StatefulSetNamespaceLister {
	return &StatefulSetNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// StatefulSetNamespaceLister implements the appsv1listers.StatefulSetNamespaceLister interface.
type StatefulSetNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all appsv1.StatefulSet in the indexer for a given namespace.
func (s StatefulSetNamespaceLister) List(selector labels.Selector) (ret []*appsv1.StatefulSet, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*appsv1.StatefulSet)
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

// Get retrieves the appsv1.StatefulSet from the indexer for a given namespace and name.
func (s StatefulSetNamespaceLister) Get(name string) (*appsv1.StatefulSet, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(appsv1.Resource("StatefulSet"), name)
	}
	return obj.(*appsv1.StatefulSet), nil
}
