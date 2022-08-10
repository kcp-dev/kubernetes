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
	"github.com/kcp-dev/logicalcluster"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
)

var _ appsv1listers.ControllerRevisionLister = &ControllerRevisionClusterLister{}

// ControllerRevisionClusterLister implements the appsv1listers.ControllerRevisionLister interface.
type ControllerRevisionClusterLister struct {
	indexer cache.Indexer
}

// NewControllerRevisionClusterLister returns a new ControllerRevisionClusterLister.
func NewControllerRevisionClusterLister(indexer cache.Indexer) appsv1listers.ControllerRevisionLister {
	return &ControllerRevisionClusterLister{indexer: indexer}
}

// List lists all appsv1.ControllerRevision in the indexer.
func (s ControllerRevisionClusterLister) List(selector labels.Selector) (ret []*appsv1.ControllerRevision, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*appsv1.ControllerRevision))
	})
	return ret, err
}

// ControllerRevisions returns an object that can list and get appsv1.ControllerRevision.
func (s ControllerRevisionClusterLister) ControllerRevisions(namespace string) appsv1listers.ControllerRevisionNamespaceLister {
	panic("Calling 'ControllerRevisions' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get appsv1.ControllerRevision.

func (s ControllerRevisionClusterLister) Cluster(cluster logicalcluster.Name) appsv1listers.ControllerRevisionLister {
	return &ControllerRevisionLister{indexer: s.indexer, cluster: cluster}
}

// ControllerRevisionLister implements the appsv1listers.ControllerRevisionLister interface.
type ControllerRevisionLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all appsv1.ControllerRevision in the indexer.
func (s ControllerRevisionLister) List(selector labels.Selector) (ret []*appsv1.ControllerRevision, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*appsv1.ControllerRevision)
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

// ControllerRevisions returns an object that can list and get appsv1.ControllerRevision.
func (s ControllerRevisionLister) ControllerRevisions(namespace string) appsv1listers.ControllerRevisionNamespaceLister {
	return &ControllerRevisionNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// ControllerRevisionNamespaceLister implements the appsv1listers.ControllerRevisionNamespaceLister interface.
type ControllerRevisionNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all appsv1.ControllerRevision in the indexer for a given namespace.
func (s ControllerRevisionNamespaceLister) List(selector labels.Selector) (ret []*appsv1.ControllerRevision, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*appsv1.ControllerRevision)
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

// Get retrieves the appsv1.ControllerRevision from the indexer for a given namespace and name.
func (s ControllerRevisionNamespaceLister) Get(name string) (*appsv1.ControllerRevision, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(appsv1.Resource("ControllerRevision"), name)
	}
	return obj.(*appsv1.ControllerRevision), nil
}
