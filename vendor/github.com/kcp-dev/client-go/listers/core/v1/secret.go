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
	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v3"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// SecretClusterLister can list Secrets across all workspaces, or scope down to a SecretLister for one workspace.
// All objects returned here must be treated as read-only.
type SecretClusterLister interface {
	// List lists all Secrets in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*corev1.Secret, err error)
	// Cluster returns a lister that can list and get Secrets in one workspace.
	Cluster(clusterName logicalcluster.Name) corev1listers.SecretLister
	SecretClusterListerExpansion
}

type secretClusterLister struct {
	indexer cache.Indexer
}

// NewSecretClusterLister returns a new SecretClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
// - has the kcpcache.ClusterAndNamespaceIndex as an index
func NewSecretClusterLister(indexer cache.Indexer) *secretClusterLister {
	return &secretClusterLister{indexer: indexer}
}

// List lists all Secrets in the indexer across all workspaces.
func (s *secretClusterLister) List(selector labels.Selector) (ret []*corev1.Secret, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*corev1.Secret))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get Secrets.
func (s *secretClusterLister) Cluster(clusterName logicalcluster.Name) corev1listers.SecretLister {
	return &secretLister{indexer: s.indexer, clusterName: clusterName}
}

// secretLister implements the corev1listers.SecretLister interface.
type secretLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
}

// List lists all Secrets in the indexer for a workspace.
func (s *secretLister) List(selector labels.Selector) (ret []*corev1.Secret, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.clusterName, selector, func(i interface{}) {
		ret = append(ret, i.(*corev1.Secret))
	})
	return ret, err
}

// Secrets returns an object that can list and get Secrets in one namespace.
func (s *secretLister) Secrets(namespace string) corev1listers.SecretNamespaceLister {
	return &secretNamespaceLister{indexer: s.indexer, clusterName: s.clusterName, namespace: namespace}
}

// secretNamespaceLister implements the corev1listers.SecretNamespaceLister interface.
type secretNamespaceLister struct {
	indexer     cache.Indexer
	clusterName logicalcluster.Name
	namespace   string
}

// List lists all Secrets in the indexer for a given workspace and namespace.
func (s *secretNamespaceLister) List(selector labels.Selector) (ret []*corev1.Secret, err error) {
	err = kcpcache.ListAllByClusterAndNamespace(s.indexer, s.clusterName, s.namespace, selector, func(i interface{}) {
		ret = append(ret, i.(*corev1.Secret))
	})
	return ret, err
}

// Get retrieves the Secret from the indexer for a given workspace, namespace and name.
func (s *secretNamespaceLister) Get(name string) (*corev1.Secret, error) {
	key := kcpcache.ToClusterAwareKey(s.clusterName.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(corev1.Resource("secrets"), name)
	}
	return obj.(*corev1.Secret), nil
}
