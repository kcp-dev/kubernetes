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

package v1beta1

import (
	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v3"

	coordinationv1beta1 "k8s.io/api/coordination/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	coordinationv1beta1listers "k8s.io/client-go/listers/coordination/v1beta1"
	"k8s.io/client-go/tools/cache"
)

// LeaseClusterLister can list Leases across all workspaces, or scope down to a LeaseLister for one workspace.
// All objects returned here must be treated as read-only.
type LeaseClusterLister interface {
	// List lists all Leases in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*coordinationv1beta1.Lease, err error)
	// Cluster returns a lister that can list and get Leases in one workspace.
	Cluster(clusterName logicalcluster.Name) coordinationv1beta1listers.LeaseLister
	LeaseClusterListerExpansion
}

type leaseClusterLister struct {
	indexer cache.Indexer
}

// NewLeaseClusterLister returns a new LeaseClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
// - has the kcpcache.ClusterAndNamespaceIndex as an index
func NewLeaseClusterLister(indexer cache.Indexer) *leaseClusterLister {
	return &leaseClusterLister{indexer: indexer}
}

// List lists all Leases in the indexer across all workspaces.
func (s *leaseClusterLister) List(selector labels.Selector) (ret []*coordinationv1beta1.Lease, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*coordinationv1beta1.Lease))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get Leases.
func (s *leaseClusterLister) Cluster(clusterName logicalcluster.Name) coordinationv1beta1listers.LeaseLister {
	return &leaseLister{indexer: s.indexer, clusterName: clusterName}
}

// leaseLister implements the coordinationv1beta1listers.LeaseLister interface.
type leaseLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
}

// List lists all Leases in the indexer for a workspace.
func (s *leaseLister) List(selector labels.Selector) (ret []*coordinationv1beta1.Lease, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.clusterName, selector, func(i interface{}) {
		ret = append(ret, i.(*coordinationv1beta1.Lease))
	})
	return ret, err
}

// Leases returns an object that can list and get Leases in one namespace.
func (s *leaseLister) Leases(namespace string) coordinationv1beta1listers.LeaseNamespaceLister {
	return &leaseNamespaceLister{indexer: s.indexer, clusterName: s.clusterName, namespace: namespace}
}

// leaseNamespaceLister implements the coordinationv1beta1listers.LeaseNamespaceLister interface.
type leaseNamespaceLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
	namespace   string
}

// List lists all Leases in the indexer for a given workspace and namespace.
func (s *leaseNamespaceLister) List(selector labels.Selector) (ret []*coordinationv1beta1.Lease, err error) {
	err = kcpcache.ListAllByClusterAndNamespace(s.indexer, s.clusterName, s.namespace, selector, func(i interface{}) {
		ret = append(ret, i.(*coordinationv1beta1.Lease))
	})
	return ret, err
}

// Get retrieves the Lease from the indexer for a given workspace, namespace and name.
func (s *leaseNamespaceLister) Get(name string) (*coordinationv1beta1.Lease, error) {
	key := kcpcache.ToClusterAwareKey(s.clusterName.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(coordinationv1beta1.Resource("leases"), name)
	}
	return obj.(*coordinationv1beta1.Lease), nil
}
