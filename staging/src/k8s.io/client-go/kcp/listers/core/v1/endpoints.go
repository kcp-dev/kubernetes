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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

var _ corev1listers.EndpointsLister = &EndpointsClusterLister{}

// EndpointsClusterLister implements the corev1listers.EndpointsLister interface.
type EndpointsClusterLister struct {
	indexer cache.Indexer
}

// NewEndpointsClusterLister returns a new EndpointsClusterLister.
func NewEndpointsClusterLister(indexer cache.Indexer) corev1listers.EndpointsLister {
	return &EndpointsClusterLister{indexer: indexer}
}

// List lists all corev1.Endpoints in the indexer.
func (s EndpointsClusterLister) List(selector labels.Selector) (ret []*corev1.Endpoints, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*corev1.Endpoints))
	})
	return ret, err
}

// Endpoints returns an object that can list and get corev1.Endpoints.
func (s EndpointsClusterLister) Endpoints(namespace string) corev1listers.EndpointsNamespaceLister {
	panic("Calling 'Endpoints' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get corev1.Endpoints.

func (s EndpointsClusterLister) Cluster(cluster logicalcluster.Name) corev1listers.EndpointsLister {
	return &EndpointsLister{indexer: s.indexer, cluster: cluster}
}

// EndpointsLister implements the corev1listers.EndpointsLister interface.
type EndpointsLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all corev1.Endpoints in the indexer.
func (s EndpointsLister) List(selector labels.Selector) (ret []*corev1.Endpoints, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*corev1.Endpoints)
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

// Endpoints returns an object that can list and get corev1.Endpoints.
func (s EndpointsLister) Endpoints(namespace string) corev1listers.EndpointsNamespaceLister {
	return &EndpointsNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// EndpointsNamespaceLister implements the corev1listers.EndpointsNamespaceLister interface.
type EndpointsNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all corev1.Endpoints in the indexer for a given namespace.
func (s EndpointsNamespaceLister) List(selector labels.Selector) (ret []*corev1.Endpoints, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*corev1.Endpoints)
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

// Get retrieves the corev1.Endpoints from the indexer for a given namespace and name.
func (s EndpointsNamespaceLister) Get(name string) (*corev1.Endpoints, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(corev1.Resource("Endpoints"), name)
	}
	return obj.(*corev1.Endpoints), nil
}
