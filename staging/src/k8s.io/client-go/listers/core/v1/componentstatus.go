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

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clusters"
)

// ComponentStatusLister helps list ComponentStatuses.
// All objects returned here must be treated as read-only.
type ComponentStatusLister interface {
	// List lists all ComponentStatuses in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.ComponentStatus, err error)
	// ListWithContext lists all ComponentStatuses in the indexer.
	// Objects returned here must be treated as read-only.
	ListWithContext(ctx context.Context, selector labels.Selector) (ret []*v1.ComponentStatus, err error)
	// Get retrieves the ComponentStatus from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.ComponentStatus, error)
	// GetWithContext retrieves the ComponentStatus from the index for a given name.
	// Objects returned here must be treated as read-only.
	GetWithContext(ctx context.Context, name string) (*v1.ComponentStatus, error)
	ComponentStatusListerExpansion
}

// componentStatusLister implements the ComponentStatusLister interface.
type componentStatusLister struct {
	indexer cache.Indexer
}

// NewComponentStatusLister returns a new ComponentStatusLister.
func NewComponentStatusLister(indexer cache.Indexer) ComponentStatusLister {
	return &componentStatusLister{indexer: indexer}
}

// List lists all ComponentStatuses in the indexer.
func (s *componentStatusLister) List(selector labels.Selector) (ret []*v1.ComponentStatus, err error) {
	return s.ListWithContext(context.Background(), selector)
}

// ListWithContext lists all ComponentStatuses in the indexer.
func (s *componentStatusLister) ListWithContext(ctx context.Context, selector labels.Selector) (ret []*v1.ComponentStatus, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ComponentStatus))
	})
	return ret, err
}

// Get retrieves the ComponentStatus from the index for a given name.
func (s *componentStatusLister) Get(name string) (*v1.ComponentStatus, error) {
	return s.GetWithContext(context.Background(), name)
}

// GetWithContext retrieves the ComponentStatus from the index for a given name.
func (s *componentStatusLister) GetWithContext(ctx context.Context, name string) (*v1.ComponentStatus, error) {
	clusterName, objectName := clusters.SplitClusterAwareKey(name)
	if clusterName == "" {
		clusterName = "admin"
	}
	key := clusters.ToClusterAwareKey(clusterName, objectName)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("componentstatus"), name)
	}
	return obj.(*v1.ComponentStatus), nil
}
