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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

var _ corev1listers.PersistentVolumeClaimLister = &PersistentVolumeClaimClusterLister{}

// PersistentVolumeClaimClusterLister implements the corev1listers.PersistentVolumeClaimLister interface.
type PersistentVolumeClaimClusterLister struct {
	indexer cache.Indexer
}

// NewPersistentVolumeClaimClusterLister returns a new PersistentVolumeClaimClusterLister.
func NewPersistentVolumeClaimClusterLister(indexer cache.Indexer) corev1listers.PersistentVolumeClaimLister {
	return &PersistentVolumeClaimClusterLister{indexer: indexer}
}

// List lists all corev1.PersistentVolumeClaim in the indexer.
func (s PersistentVolumeClaimClusterLister) List(selector labels.Selector) (ret []*corev1.PersistentVolumeClaim, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*corev1.PersistentVolumeClaim))
	})
	return ret, err
}

// PersistentVolumeClaims returns an object that can list and get corev1.PersistentVolumeClaim.
func (s PersistentVolumeClaimClusterLister) PersistentVolumeClaims(namespace string) corev1listers.PersistentVolumeClaimNamespaceLister {
	panic("Calling 'PersistentVolumeClaims' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get corev1.PersistentVolumeClaim.

func (s PersistentVolumeClaimClusterLister) Cluster(cluster logicalcluster.Name) corev1listers.PersistentVolumeClaimLister {
	return &PersistentVolumeClaimLister{indexer: s.indexer, cluster: cluster}
}

// PersistentVolumeClaimLister implements the corev1listers.PersistentVolumeClaimLister interface.
type PersistentVolumeClaimLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all corev1.PersistentVolumeClaim in the indexer.
func (s PersistentVolumeClaimLister) List(selector labels.Selector) (ret []*corev1.PersistentVolumeClaim, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*corev1.PersistentVolumeClaim)
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

// PersistentVolumeClaims returns an object that can list and get corev1.PersistentVolumeClaim.
func (s PersistentVolumeClaimLister) PersistentVolumeClaims(namespace string) corev1listers.PersistentVolumeClaimNamespaceLister {
	return &PersistentVolumeClaimNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// PersistentVolumeClaimNamespaceLister implements the corev1listers.PersistentVolumeClaimNamespaceLister interface.
type PersistentVolumeClaimNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all corev1.PersistentVolumeClaim in the indexer for a given namespace.
func (s PersistentVolumeClaimNamespaceLister) List(selector labels.Selector) (ret []*corev1.PersistentVolumeClaim, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*corev1.PersistentVolumeClaim)
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

// Get retrieves the corev1.PersistentVolumeClaim from the indexer for a given namespace and name.
func (s PersistentVolumeClaimNamespaceLister) Get(name string) (*corev1.PersistentVolumeClaim, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(corev1.Resource("PersistentVolumeClaim"), name)
	}
	return obj.(*corev1.PersistentVolumeClaim), nil
}
