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
	apimachinerycache "github.com/kcp-dev/apimachinery/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v2"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	discoveryv1beta1listers "k8s.io/client-go/listers/discovery/v1beta1"
	"k8s.io/client-go/tools/cache"
)

var _ discoveryv1beta1listers.EndpointSliceLister = &EndpointSliceClusterLister{}

// EndpointSliceClusterLister implements the discoveryv1beta1listers.EndpointSliceLister interface.
type EndpointSliceClusterLister struct {
	indexer cache.Indexer
}

// NewEndpointSliceClusterLister returns a new EndpointSliceClusterLister.
func NewEndpointSliceClusterLister(indexer cache.Indexer) discoveryv1beta1listers.EndpointSliceLister {
	return &EndpointSliceClusterLister{indexer: indexer}
}

// List lists all discoveryv1beta1.EndpointSlice in the indexer.
func (s EndpointSliceClusterLister) List(selector labels.Selector) (ret []*discoveryv1beta1.EndpointSlice, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*discoveryv1beta1.EndpointSlice))
	})
	return ret, err
}

// EndpointSlices returns an object that can list and get discoveryv1beta1.EndpointSlice.
func (s EndpointSliceClusterLister) EndpointSlices(namespace string) discoveryv1beta1listers.EndpointSliceNamespaceLister {
	panic("Calling 'EndpointSlices' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get discoveryv1beta1.EndpointSlice.

func (s EndpointSliceClusterLister) Cluster(cluster logicalcluster.Name) discoveryv1beta1listers.EndpointSliceLister {
	return &EndpointSliceLister{indexer: s.indexer, cluster: cluster}
}

// EndpointSliceLister implements the discoveryv1beta1listers.EndpointSliceLister interface.
type EndpointSliceLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all discoveryv1beta1.EndpointSlice in the indexer.
func (s EndpointSliceLister) List(selector labels.Selector) (ret []*discoveryv1beta1.EndpointSlice, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*discoveryv1beta1.EndpointSlice)
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

// EndpointSlices returns an object that can list and get discoveryv1beta1.EndpointSlice.
func (s EndpointSliceLister) EndpointSlices(namespace string) discoveryv1beta1listers.EndpointSliceNamespaceLister {
	return &EndpointSliceNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// EndpointSliceNamespaceLister implements the discoveryv1beta1listers.EndpointSliceNamespaceLister interface.
type EndpointSliceNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all discoveryv1beta1.EndpointSlice in the indexer for a given namespace.
func (s EndpointSliceNamespaceLister) List(selector labels.Selector) (ret []*discoveryv1beta1.EndpointSlice, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*discoveryv1beta1.EndpointSlice)
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

// Get retrieves the discoveryv1beta1.EndpointSlice from the indexer for a given namespace and name.
func (s EndpointSliceNamespaceLister) Get(name string) (*discoveryv1beta1.EndpointSlice, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(discoveryv1beta1.Resource("EndpointSlice"), name)
	}
	return obj.(*discoveryv1beta1.EndpointSlice), nil
}
