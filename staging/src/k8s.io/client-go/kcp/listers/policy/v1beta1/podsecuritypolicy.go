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
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	policyv1beta1listers "k8s.io/client-go/listers/policy/v1beta1"
	"k8s.io/client-go/tools/cache"
)

var _ policyv1beta1listers.PodSecurityPolicyLister = &PodSecurityPolicyClusterLister{}

// PodSecurityPolicyClusterLister implements the policyv1beta1listers.PodSecurityPolicyLister interface.
type PodSecurityPolicyClusterLister struct {
	indexer cache.Indexer
}

// NewPodSecurityPolicyClusterLister returns a new PodSecurityPolicyClusterLister.
func NewPodSecurityPolicyClusterLister(indexer cache.Indexer) policyv1beta1listers.PodSecurityPolicyLister {
	return &PodSecurityPolicyClusterLister{indexer: indexer}
}

// List lists all policyv1beta1.PodSecurityPolicy in the indexer.
func (s PodSecurityPolicyClusterLister) List(selector labels.Selector) (ret []*policyv1beta1.PodSecurityPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*policyv1beta1.PodSecurityPolicy))
	})
	return ret, err
}

// Get retrieves the policyv1beta1.PodSecurityPolicy from the indexer for a given name.
func (s PodSecurityPolicyClusterLister) Get(name string) (*policyv1beta1.PodSecurityPolicy, error) {
	panic("Calling 'Get' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get policyv1beta1.PodSecurityPolicy.

func (s PodSecurityPolicyClusterLister) Cluster(cluster logicalcluster.Name) policyv1beta1listers.PodSecurityPolicyLister {
	return &PodSecurityPolicyLister{indexer: s.indexer, cluster: cluster}
}

// PodSecurityPolicyLister implements the policyv1beta1listers.PodSecurityPolicyLister interface.
type PodSecurityPolicyLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all policyv1beta1.PodSecurityPolicy in the indexer.
func (s PodSecurityPolicyLister) List(selector labels.Selector) (ret []*policyv1beta1.PodSecurityPolicy, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*policyv1beta1.PodSecurityPolicy)
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

// Get retrieves the policyv1beta1.PodSecurityPolicy from the indexer for a given name.
func (s PodSecurityPolicyLister) Get(name string) (*policyv1beta1.PodSecurityPolicy, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(policyv1beta1.Resource("PodSecurityPolicy"), name)
	}
	return obj.(*policyv1beta1.PodSecurityPolicy), nil
}
