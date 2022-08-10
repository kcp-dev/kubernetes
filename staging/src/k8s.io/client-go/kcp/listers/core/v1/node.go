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

var _ corev1listers.NodeLister = &NodeClusterLister{}

// NodeClusterLister implements the corev1listers.NodeLister interface.
type NodeClusterLister struct {
	indexer cache.Indexer
}

// NewNodeClusterLister returns a new NodeClusterLister.
func NewNodeClusterLister(indexer cache.Indexer) corev1listers.NodeLister {
	return &NodeClusterLister{indexer: indexer}
}

// List lists all corev1.Node in the indexer.
func (s NodeClusterLister) List(selector labels.Selector) (ret []*corev1.Node, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*corev1.Node))
	})
	return ret, err
}

// Get retrieves the corev1.Node from the indexer for a given name.
func (s NodeClusterLister) Get(name string) (*corev1.Node, error) {
	panic("Calling 'Get' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get corev1.Node.

func (s NodeClusterLister) Cluster(cluster logicalcluster.Name) corev1listers.NodeLister {
	return &NodeLister{indexer: s.indexer, cluster: cluster}
}

// NodeLister implements the corev1listers.NodeLister interface.
type NodeLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all corev1.Node in the indexer.
func (s NodeLister) List(selector labels.Selector) (ret []*corev1.Node, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*corev1.Node)
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

// Get retrieves the corev1.Node from the indexer for a given name.
func (s NodeLister) Get(name string) (*corev1.Node, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(corev1.Resource("Node"), name)
	}
	return obj.(*corev1.Node), nil
}
