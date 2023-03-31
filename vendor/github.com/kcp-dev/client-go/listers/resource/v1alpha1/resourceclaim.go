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

package v1alpha1

import (
	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v3"

	resourcev1alpha1 "k8s.io/api/resource/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	resourcev1alpha1listers "k8s.io/client-go/listers/resource/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

// ResourceClaimClusterLister can list ResourceClaims across all workspaces, or scope down to a ResourceClaimLister for one workspace.
// All objects returned here must be treated as read-only.
type ResourceClaimClusterLister interface {
	// List lists all ResourceClaims in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*resourcev1alpha1.ResourceClaim, err error)
	// Cluster returns a lister that can list and get ResourceClaims in one workspace.
	Cluster(clusterName logicalcluster.Name) resourcev1alpha1listers.ResourceClaimLister
	ResourceClaimClusterListerExpansion
}

type resourceClaimClusterLister struct {
	indexer cache.Indexer
}

// NewResourceClaimClusterLister returns a new ResourceClaimClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
// - has the kcpcache.ClusterAndNamespaceIndex as an index
func NewResourceClaimClusterLister(indexer cache.Indexer) *resourceClaimClusterLister {
	return &resourceClaimClusterLister{indexer: indexer}
}

// List lists all ResourceClaims in the indexer across all workspaces.
func (s *resourceClaimClusterLister) List(selector labels.Selector) (ret []*resourcev1alpha1.ResourceClaim, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*resourcev1alpha1.ResourceClaim))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get ResourceClaims.
func (s *resourceClaimClusterLister) Cluster(clusterName logicalcluster.Name) resourcev1alpha1listers.ResourceClaimLister {
	return &resourceClaimLister{indexer: s.indexer, clusterName: clusterName}
}

// resourceClaimLister implements the resourcev1alpha1listers.ResourceClaimLister interface.
type resourceClaimLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
}

// List lists all ResourceClaims in the indexer for a workspace.
func (s *resourceClaimLister) List(selector labels.Selector) (ret []*resourcev1alpha1.ResourceClaim, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.clusterName, selector, func(i interface{}) {
		ret = append(ret, i.(*resourcev1alpha1.ResourceClaim))
	})
	return ret, err
}

// ResourceClaims returns an object that can list and get ResourceClaims in one namespace.
func (s *resourceClaimLister) ResourceClaims(namespace string) resourcev1alpha1listers.ResourceClaimNamespaceLister {
	return &resourceClaimNamespaceLister{indexer: s.indexer, clusterName: s.clusterName, namespace: namespace}
}

// resourceClaimNamespaceLister implements the resourcev1alpha1listers.ResourceClaimNamespaceLister interface.
type resourceClaimNamespaceLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
	namespace   string
}

// List lists all ResourceClaims in the indexer for a given workspace and namespace.
func (s *resourceClaimNamespaceLister) List(selector labels.Selector) (ret []*resourcev1alpha1.ResourceClaim, err error) {
	err = kcpcache.ListAllByClusterAndNamespace(s.indexer, s.clusterName, s.namespace, selector, func(i interface{}) {
		ret = append(ret, i.(*resourcev1alpha1.ResourceClaim))
	})
	return ret, err
}

// Get retrieves the ResourceClaim from the indexer for a given workspace, namespace and name.
func (s *resourceClaimNamespaceLister) Get(name string) (*resourcev1alpha1.ResourceClaim, error) {
	key := kcpcache.ToClusterAwareKey(s.clusterName.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(resourcev1alpha1.Resource("resourceclaims"), name)
	}
	return obj.(*resourcev1alpha1.ResourceClaim), nil
}
