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
	flowcontrolv1alpha1 "k8s.io/api/flowcontrol/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	flowcontrolv1alpha1listers "k8s.io/client-go/listers/flowcontrol/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

var _ flowcontrolv1alpha1listers.PriorityLevelConfigurationLister = &PriorityLevelConfigurationClusterLister{}

// PriorityLevelConfigurationClusterLister implements the flowcontrolv1alpha1listers.PriorityLevelConfigurationLister interface.
type PriorityLevelConfigurationClusterLister struct {
	indexer cache.Indexer
}

// NewPriorityLevelConfigurationClusterLister returns a new PriorityLevelConfigurationClusterLister.
func NewPriorityLevelConfigurationClusterLister(indexer cache.Indexer) flowcontrolv1alpha1listers.PriorityLevelConfigurationLister {
	return &PriorityLevelConfigurationClusterLister{indexer: indexer}
}

// List lists all flowcontrolv1alpha1.PriorityLevelConfiguration in the indexer.
func (s PriorityLevelConfigurationClusterLister) List(selector labels.Selector) (ret []*flowcontrolv1alpha1.PriorityLevelConfiguration, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*flowcontrolv1alpha1.PriorityLevelConfiguration))
	})
	return ret, err
}

// Get retrieves the flowcontrolv1alpha1.PriorityLevelConfiguration from the indexer for a given name.
func (s PriorityLevelConfigurationClusterLister) Get(name string) (*flowcontrolv1alpha1.PriorityLevelConfiguration, error) {
	panic("Calling 'Get' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get flowcontrolv1alpha1.PriorityLevelConfiguration.

func (s PriorityLevelConfigurationClusterLister) Cluster(cluster logicalcluster.Name) flowcontrolv1alpha1listers.PriorityLevelConfigurationLister {
	return &PriorityLevelConfigurationLister{indexer: s.indexer, cluster: cluster}
}

// PriorityLevelConfigurationLister implements the flowcontrolv1alpha1listers.PriorityLevelConfigurationLister interface.
type PriorityLevelConfigurationLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all flowcontrolv1alpha1.PriorityLevelConfiguration in the indexer.
func (s PriorityLevelConfigurationLister) List(selector labels.Selector) (ret []*flowcontrolv1alpha1.PriorityLevelConfiguration, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*flowcontrolv1alpha1.PriorityLevelConfiguration)
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

// Get retrieves the flowcontrolv1alpha1.PriorityLevelConfiguration from the indexer for a given name.
func (s PriorityLevelConfigurationLister) Get(name string) (*flowcontrolv1alpha1.PriorityLevelConfiguration, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(flowcontrolv1alpha1.Resource("PriorityLevelConfiguration"), name)
	}
	return obj.(*flowcontrolv1alpha1.PriorityLevelConfiguration), nil
}
