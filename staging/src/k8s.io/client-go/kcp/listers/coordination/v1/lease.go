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
	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	coordinationv1listers "k8s.io/client-go/listers/coordination/v1"
	"k8s.io/client-go/tools/cache"
)

var _ coordinationv1listers.LeaseLister = &LeaseClusterLister{}

// LeaseClusterLister implements the coordinationv1listers.LeaseLister interface.
type LeaseClusterLister struct {
	indexer cache.Indexer
}

// NewLeaseClusterLister returns a new LeaseClusterLister.
func NewLeaseClusterLister(indexer cache.Indexer) coordinationv1listers.LeaseLister {
	return &LeaseClusterLister{indexer: indexer}
}

// List lists all coordinationv1.Lease in the indexer.
func (s LeaseClusterLister) List(selector labels.Selector) (ret []*coordinationv1.Lease, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*coordinationv1.Lease))
	})
	return ret, err
}

// Leases returns an object that can list and get coordinationv1.Lease.
func (s LeaseClusterLister) Leases(namespace string) coordinationv1listers.LeaseNamespaceLister {
	panic("Calling 'Leases' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get coordinationv1.Lease.

func (s LeaseClusterLister) Cluster(cluster logicalcluster.Name) coordinationv1listers.LeaseLister {
	return &LeaseLister{indexer: s.indexer, cluster: cluster}
}

// LeaseLister implements the coordinationv1listers.LeaseLister interface.
type LeaseLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all coordinationv1.Lease in the indexer.
func (s LeaseLister) List(selector labels.Selector) (ret []*coordinationv1.Lease, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*coordinationv1.Lease)
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

// Leases returns an object that can list and get coordinationv1.Lease.
func (s LeaseLister) Leases(namespace string) coordinationv1listers.LeaseNamespaceLister {
	return &LeaseNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// LeaseNamespaceLister implements the coordinationv1listers.LeaseNamespaceLister interface.
type LeaseNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all coordinationv1.Lease in the indexer for a given namespace.
func (s LeaseNamespaceLister) List(selector labels.Selector) (ret []*coordinationv1.Lease, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*coordinationv1.Lease)
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

// Get retrieves the coordinationv1.Lease from the indexer for a given namespace and name.
func (s LeaseNamespaceLister) Get(name string) (*coordinationv1.Lease, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(coordinationv1.Resource("Lease"), name)
	}
	return obj.(*coordinationv1.Lease), nil
}
