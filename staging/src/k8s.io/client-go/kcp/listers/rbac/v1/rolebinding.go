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
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
)

var _ rbacv1listers.RoleBindingLister = &RoleBindingClusterLister{}

// RoleBindingClusterLister implements the rbacv1listers.RoleBindingLister interface.
type RoleBindingClusterLister struct {
	indexer cache.Indexer
}

// NewRoleBindingClusterLister returns a new RoleBindingClusterLister.
func NewRoleBindingClusterLister(indexer cache.Indexer) rbacv1listers.RoleBindingLister {
	return &RoleBindingClusterLister{indexer: indexer}
}

// List lists all rbacv1.RoleBinding in the indexer.
func (s RoleBindingClusterLister) List(selector labels.Selector) (ret []*rbacv1.RoleBinding, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*rbacv1.RoleBinding))
	})
	return ret, err
}

// RoleBindings returns an object that can list and get rbacv1.RoleBinding.
func (s RoleBindingClusterLister) RoleBindings(namespace string) rbacv1listers.RoleBindingNamespaceLister {
	panic("Calling 'RoleBindings' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get rbacv1.RoleBinding.

func (s RoleBindingClusterLister) Cluster(cluster logicalcluster.Name) rbacv1listers.RoleBindingLister {
	return &RoleBindingLister{indexer: s.indexer, cluster: cluster}
}

// RoleBindingLister implements the rbacv1listers.RoleBindingLister interface.
type RoleBindingLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all rbacv1.RoleBinding in the indexer.
func (s RoleBindingLister) List(selector labels.Selector) (ret []*rbacv1.RoleBinding, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*rbacv1.RoleBinding)
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

// RoleBindings returns an object that can list and get rbacv1.RoleBinding.
func (s RoleBindingLister) RoleBindings(namespace string) rbacv1listers.RoleBindingNamespaceLister {
	return &RoleBindingNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// RoleBindingNamespaceLister implements the rbacv1listers.RoleBindingNamespaceLister interface.
type RoleBindingNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all rbacv1.RoleBinding in the indexer for a given namespace.
func (s RoleBindingNamespaceLister) List(selector labels.Selector) (ret []*rbacv1.RoleBinding, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*rbacv1.RoleBinding)
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

// Get retrieves the rbacv1.RoleBinding from the indexer for a given namespace and name.
func (s RoleBindingNamespaceLister) Get(name string) (*rbacv1.RoleBinding, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(rbacv1.Resource("RoleBinding"), name)
	}
	return obj.(*rbacv1.RoleBinding), nil
}
