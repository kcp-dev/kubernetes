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

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clusters"
)

// IngressClassLister helps list IngressClasses.
// All objects returned here must be treated as read-only.
type IngressClassLister interface {
	// List lists all IngressClasses in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.IngressClass, err error)
	// ListWithContext lists all IngressClasses in the indexer.
	// Objects returned here must be treated as read-only.
	ListWithContext(ctx context.Context, selector labels.Selector) (ret []*v1.IngressClass, err error)
	// Get retrieves the IngressClass from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.IngressClass, error)
	// GetWithContext retrieves the IngressClass from the index for a given name.
	// Objects returned here must be treated as read-only.
	GetWithContext(ctx context.Context, name string) (*v1.IngressClass, error)
	IngressClassListerExpansion
}

// ingressClassLister implements the IngressClassLister interface.
type ingressClassLister struct {
	indexer cache.Indexer
}

// NewIngressClassLister returns a new IngressClassLister.
func NewIngressClassLister(indexer cache.Indexer) IngressClassLister {
	return &ingressClassLister{indexer: indexer}
}

// List lists all IngressClasses in the indexer.
func (s *ingressClassLister) List(selector labels.Selector) (ret []*v1.IngressClass, err error) {
	return s.ListWithContext(context.Background(), selector)
}

// ListWithContext lists all IngressClasses in the indexer.
func (s *ingressClassLister) ListWithContext(ctx context.Context, selector labels.Selector) (ret []*v1.IngressClass, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.IngressClass))
	})
	return ret, err
}

// Get retrieves the IngressClass from the index for a given name.
func (s *ingressClassLister) Get(name string) (*v1.IngressClass, error) {
	return s.GetWithContext(context.Background(), name)
}

// GetWithContext retrieves the IngressClass from the index for a given name.
func (s *ingressClassLister) GetWithContext(ctx context.Context, name string) (*v1.IngressClass, error) {
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
		return nil, errors.NewNotFound(v1.Resource("ingressclass"), name)
	}
	return obj.(*v1.IngressClass), nil
}
