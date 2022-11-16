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

package v1

import (
	kcpcache "github.com/kcp-dev/apimachinery/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v2"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
)

// ClusterRoleBindingClusterLister can list ClusterRoleBindings across all workspaces, or scope down to a ClusterRoleBindingLister for one workspace.
type ClusterRoleBindingClusterLister interface {
	List(selector labels.Selector) (ret []*rbacv1.ClusterRoleBinding, err error)
	Cluster(cluster logicalcluster.Name) rbacv1listers.ClusterRoleBindingLister
}

type clusterRoleBindingClusterLister struct {
	indexer cache.Indexer
}

// NewClusterRoleBindingClusterLister returns a new ClusterRoleBindingClusterLister.
func NewClusterRoleBindingClusterLister(indexer cache.Indexer) *clusterRoleBindingClusterLister {
	return &clusterRoleBindingClusterLister{indexer: indexer}
}

// List lists all ClusterRoleBindings in the indexer across all workspaces.
func (s *clusterRoleBindingClusterLister) List(selector labels.Selector) (ret []*rbacv1.ClusterRoleBinding, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*rbacv1.ClusterRoleBinding))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get ClusterRoleBindings.
func (s *clusterRoleBindingClusterLister) Cluster(cluster logicalcluster.Name) rbacv1listers.ClusterRoleBindingLister {
	return &clusterRoleBindingLister{indexer: s.indexer, cluster: cluster}
}

// clusterRoleBindingLister implements the rbacv1listers.ClusterRoleBindingLister interface.
type clusterRoleBindingLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all ClusterRoleBindings in the indexer for a workspace.
func (s *clusterRoleBindingLister) List(selector labels.Selector) (ret []*rbacv1.ClusterRoleBinding, err error) {
	selectAll := selector == nil || selector.Empty()

	list, err := s.indexer.ByIndex(kcpcache.ClusterIndexName, kcpcache.ClusterIndexKey(s.cluster))
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*rbacv1.ClusterRoleBinding)
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

// Get retrieves the ClusterRoleBinding from the indexer for a given workspace and name.
func (s *clusterRoleBindingLister) Get(name string) (*rbacv1.ClusterRoleBinding, error) {
	key := kcpcache.ToClusterAwareKey(s.cluster.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(rbacv1.Resource("ClusterRoleBinding"), name)
	}
	return obj.(*rbacv1.ClusterRoleBinding), nil
}
