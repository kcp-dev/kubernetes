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

package v1beta2

import (
	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v3"

	flowcontrolv1beta2 "k8s.io/api/flowcontrol/v1beta2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	flowcontrolv1beta2listers "k8s.io/client-go/listers/flowcontrol/v1beta2"
	"k8s.io/client-go/tools/cache"
)

// PriorityLevelConfigurationClusterLister can list PriorityLevelConfigurations across all workspaces, or scope down to a PriorityLevelConfigurationLister for one workspace.
// All objects returned here must be treated as read-only.
type PriorityLevelConfigurationClusterLister interface {
	// List lists all PriorityLevelConfigurations in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*flowcontrolv1beta2.PriorityLevelConfiguration, err error)
	// Cluster returns a lister that can list and get PriorityLevelConfigurations in one workspace.
	Cluster(clusterName logicalcluster.Name) flowcontrolv1beta2listers.PriorityLevelConfigurationLister
	PriorityLevelConfigurationClusterListerExpansion
}

type priorityLevelConfigurationClusterLister struct {
	indexer cache.Indexer
}

// NewPriorityLevelConfigurationClusterLister returns a new PriorityLevelConfigurationClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
func NewPriorityLevelConfigurationClusterLister(indexer cache.Indexer) *priorityLevelConfigurationClusterLister {
	return &priorityLevelConfigurationClusterLister{indexer: indexer}
}

// List lists all PriorityLevelConfigurations in the indexer across all workspaces.
func (s *priorityLevelConfigurationClusterLister) List(selector labels.Selector) (ret []*flowcontrolv1beta2.PriorityLevelConfiguration, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*flowcontrolv1beta2.PriorityLevelConfiguration))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get PriorityLevelConfigurations.
func (s *priorityLevelConfigurationClusterLister) Cluster(clusterName logicalcluster.Name) flowcontrolv1beta2listers.PriorityLevelConfigurationLister {
	return &priorityLevelConfigurationLister{indexer: s.indexer, clusterName: clusterName}
}

// priorityLevelConfigurationLister implements the flowcontrolv1beta2listers.PriorityLevelConfigurationLister interface.
type priorityLevelConfigurationLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
}

// List lists all PriorityLevelConfigurations in the indexer for a workspace.
func (s *priorityLevelConfigurationLister) List(selector labels.Selector) (ret []*flowcontrolv1beta2.PriorityLevelConfiguration, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.clusterName, selector, func(i interface{}) {
		ret = append(ret, i.(*flowcontrolv1beta2.PriorityLevelConfiguration))
	})
	return ret, err
}

// Get retrieves the PriorityLevelConfiguration from the indexer for a given workspace and name.
func (s *priorityLevelConfigurationLister) Get(name string) (*flowcontrolv1beta2.PriorityLevelConfiguration, error) {
	key := kcpcache.ToClusterAwareKey(s.clusterName.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(flowcontrolv1beta2.Resource("prioritylevelconfigurations"), name)
	}
	return obj.(*flowcontrolv1beta2.PriorityLevelConfiguration), nil
}
