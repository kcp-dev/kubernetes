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
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	networkingv1beta1listers "k8s.io/client-go/listers/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

var _ networkingv1beta1listers.IngressClassLister = &IngressClassClusterLister{}

// IngressClassClusterLister implements the networkingv1beta1listers.IngressClassLister interface.
type IngressClassClusterLister struct {
	indexer cache.Indexer
}

// NewIngressClassClusterLister returns a new IngressClassClusterLister.
func NewIngressClassClusterLister(indexer cache.Indexer) networkingv1beta1listers.IngressClassLister {
	return &IngressClassClusterLister{indexer: indexer}
}

// List lists all networkingv1beta1.IngressClass in the indexer.
func (s IngressClassClusterLister) List(selector labels.Selector) (ret []*networkingv1beta1.IngressClass, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*networkingv1beta1.IngressClass))
	})
	return ret, err
}

// Get retrieves the networkingv1beta1.IngressClass from the indexer for a given name.
func (s IngressClassClusterLister) Get(name string) (*networkingv1beta1.IngressClass, error) {
	panic("Calling 'Get' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get networkingv1beta1.IngressClass.

func (s IngressClassClusterLister) Cluster(cluster logicalcluster.Name) networkingv1beta1listers.IngressClassLister {
	return &IngressClassLister{indexer: s.indexer, cluster: cluster}
}

// IngressClassLister implements the networkingv1beta1listers.IngressClassLister interface.
type IngressClassLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all networkingv1beta1.IngressClass in the indexer.
func (s IngressClassLister) List(selector labels.Selector) (ret []*networkingv1beta1.IngressClass, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*networkingv1beta1.IngressClass)
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

// Get retrieves the networkingv1beta1.IngressClass from the indexer for a given name.
func (s IngressClassLister) Get(name string) (*networkingv1beta1.IngressClass, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(networkingv1beta1.Resource("IngressClass"), name)
	}
	return obj.(*networkingv1beta1.IngressClass), nil
}
