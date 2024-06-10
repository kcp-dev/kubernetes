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

package v1alpha2

import (
	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v3"

	resourcev1alpha2 "k8s.io/api/resource/v1alpha2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	resourcev1alpha2listers "k8s.io/client-go/listers/resource/v1alpha2"
	"k8s.io/client-go/tools/cache"
)

// ResourceClaimTemplateClusterLister can list ResourceClaimTemplates across all workspaces, or scope down to a ResourceClaimTemplateLister for one workspace.
// All objects returned here must be treated as read-only.
type ResourceClaimTemplateClusterLister interface {
	// List lists all ResourceClaimTemplates in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*resourcev1alpha2.ResourceClaimTemplate, err error)
	// Cluster returns a lister that can list and get ResourceClaimTemplates in one workspace.
	Cluster(clusterName logicalcluster.Name) resourcev1alpha2listers.ResourceClaimTemplateLister
	ResourceClaimTemplateClusterListerExpansion
}

type resourceClaimTemplateClusterLister struct {
	indexer cache.Indexer
}

// NewResourceClaimTemplateClusterLister returns a new ResourceClaimTemplateClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
// - has the kcpcache.ClusterAndNamespaceIndex as an index
func NewResourceClaimTemplateClusterLister(indexer cache.Indexer) *resourceClaimTemplateClusterLister {
	return &resourceClaimTemplateClusterLister{indexer: indexer}
}

// List lists all ResourceClaimTemplates in the indexer across all workspaces.
func (s *resourceClaimTemplateClusterLister) List(selector labels.Selector) (ret []*resourcev1alpha2.ResourceClaimTemplate, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*resourcev1alpha2.ResourceClaimTemplate))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get ResourceClaimTemplates.
func (s *resourceClaimTemplateClusterLister) Cluster(clusterName logicalcluster.Name) resourcev1alpha2listers.ResourceClaimTemplateLister {
	return &resourceClaimTemplateLister{indexer: s.indexer, clusterName: clusterName}
}

// resourceClaimTemplateLister implements the resourcev1alpha2listers.ResourceClaimTemplateLister interface.
type resourceClaimTemplateLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
}

// List lists all ResourceClaimTemplates in the indexer for a workspace.
func (s *resourceClaimTemplateLister) List(selector labels.Selector) (ret []*resourcev1alpha2.ResourceClaimTemplate, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.clusterName, selector, func(i interface{}) {
		ret = append(ret, i.(*resourcev1alpha2.ResourceClaimTemplate))
	})
	return ret, err
}

// ResourceClaimTemplates returns an object that can list and get ResourceClaimTemplates in one namespace.
func (s *resourceClaimTemplateLister) ResourceClaimTemplates(namespace string) resourcev1alpha2listers.ResourceClaimTemplateNamespaceLister {
	return &resourceClaimTemplateNamespaceLister{indexer: s.indexer, clusterName: s.clusterName, namespace: namespace}
}

// resourceClaimTemplateNamespaceLister implements the resourcev1alpha2listers.ResourceClaimTemplateNamespaceLister interface.
type resourceClaimTemplateNamespaceLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
	namespace   string
}

// List lists all ResourceClaimTemplates in the indexer for a given workspace and namespace.
func (s *resourceClaimTemplateNamespaceLister) List(selector labels.Selector) (ret []*resourcev1alpha2.ResourceClaimTemplate, err error) {
	err = kcpcache.ListAllByClusterAndNamespace(s.indexer, s.clusterName, s.namespace, selector, func(i interface{}) {
		ret = append(ret, i.(*resourcev1alpha2.ResourceClaimTemplate))
	})
	return ret, err
}

// Get retrieves the ResourceClaimTemplate from the indexer for a given workspace, namespace and name.
func (s *resourceClaimTemplateNamespaceLister) Get(name string) (*resourcev1alpha2.ResourceClaimTemplate, error) {
	key := kcpcache.ToClusterAwareKey(s.clusterName.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(resourcev1alpha2.Resource("resourceclaimtemplates"), name)
	}
	return obj.(*resourcev1alpha2.ResourceClaimTemplate), nil
}
