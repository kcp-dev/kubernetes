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

package v1alpha1

import (
	apimachinerycache "github.com/kcp-dev/apimachinery/pkg/cache"
	"github.com/kcp-dev/logicalcluster"
	rbacv1alpha1 "k8s.io/api/rbac/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	rbacv1alpha1listers "k8s.io/client-go/listers/rbac/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

var _ rbacv1alpha1listers.ClusterRoleBindingLister = &ClusterRoleBindingClusterLister{}

// ClusterRoleBindingClusterLister implements the rbacv1alpha1listers.ClusterRoleBindingLister interface.
type ClusterRoleBindingClusterLister struct {
	indexer cache.Indexer
}

// NewClusterRoleBindingClusterLister returns a new ClusterRoleBindingClusterLister.
func NewClusterRoleBindingClusterLister(indexer cache.Indexer) rbacv1alpha1listers.ClusterRoleBindingLister {
	return &ClusterRoleBindingClusterLister{indexer: indexer}
}

// List lists all rbacv1alpha1.ClusterRoleBinding in the indexer.
func (s ClusterRoleBindingClusterLister) List(selector labels.Selector) (ret []*rbacv1alpha1.ClusterRoleBinding, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*rbacv1alpha1.ClusterRoleBinding))
	})
	return ret, err
}

// Get retrieves the rbacv1alpha1.ClusterRoleBinding from the indexer for a given name.
func (s ClusterRoleBindingClusterLister) Get(name string) (*rbacv1alpha1.ClusterRoleBinding, error) {
	panic("Calling 'Get' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get rbacv1alpha1.ClusterRoleBinding.

func (s ClusterRoleBindingClusterLister) Cluster(cluster logicalcluster.Name) rbacv1alpha1listers.ClusterRoleBindingLister {
	return &ClusterRoleBindingLister{indexer: s.indexer, cluster: cluster}
}

// ClusterRoleBindingLister implements the rbacv1alpha1listers.ClusterRoleBindingLister interface.
type ClusterRoleBindingLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all rbacv1alpha1.ClusterRoleBinding in the indexer.
func (s ClusterRoleBindingLister) List(selector labels.Selector) (ret []*rbacv1alpha1.ClusterRoleBinding, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*rbacv1alpha1.ClusterRoleBinding)
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

// Get retrieves the rbacv1alpha1.ClusterRoleBinding from the indexer for a given name.
func (s ClusterRoleBindingLister) Get(name string) (*rbacv1alpha1.ClusterRoleBinding, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(rbacv1alpha1.Resource("ClusterRoleBinding"), name)
	}
	return obj.(*rbacv1alpha1.ClusterRoleBinding), nil
}
