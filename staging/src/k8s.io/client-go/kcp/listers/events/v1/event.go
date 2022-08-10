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
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	eventsv1listers "k8s.io/client-go/listers/events/v1"
	"k8s.io/client-go/tools/cache"
)

var _ eventsv1listers.EventLister = &EventClusterLister{}

// EventClusterLister implements the eventsv1listers.EventLister interface.
type EventClusterLister struct {
	indexer cache.Indexer
}

// NewEventClusterLister returns a new EventClusterLister.
func NewEventClusterLister(indexer cache.Indexer) eventsv1listers.EventLister {
	return &EventClusterLister{indexer: indexer}
}

// List lists all eventsv1.Event in the indexer.
func (s EventClusterLister) List(selector labels.Selector) (ret []*eventsv1.Event, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*eventsv1.Event))
	})
	return ret, err
}

// Events returns an object that can list and get eventsv1.Event.
func (s EventClusterLister) Events(namespace string) eventsv1listers.EventNamespaceLister {
	panic("Calling 'Events' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get eventsv1.Event.

func (s EventClusterLister) Cluster(cluster logicalcluster.Name) eventsv1listers.EventLister {
	return &EventLister{indexer: s.indexer, cluster: cluster}
}

// EventLister implements the eventsv1listers.EventLister interface.
type EventLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all eventsv1.Event in the indexer.
func (s EventLister) List(selector labels.Selector) (ret []*eventsv1.Event, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*eventsv1.Event)
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

// Events returns an object that can list and get eventsv1.Event.
func (s EventLister) Events(namespace string) eventsv1listers.EventNamespaceLister {
	return &EventNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// EventNamespaceLister implements the eventsv1listers.EventNamespaceLister interface.
type EventNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all eventsv1.Event in the indexer for a given namespace.
func (s EventNamespaceLister) List(selector labels.Selector) (ret []*eventsv1.Event, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*eventsv1.Event)
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

// Get retrieves the eventsv1.Event from the indexer for a given namespace and name.
func (s EventNamespaceLister) Get(name string) (*eventsv1.Event, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(eventsv1.Resource("Event"), name)
	}
	return obj.(*eventsv1.Event), nil
}
