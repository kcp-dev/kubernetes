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
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	batchv1beta1listers "k8s.io/client-go/listers/batch/v1beta1"
	"k8s.io/client-go/tools/cache"
)

var _ batchv1beta1listers.CronJobLister = &CronJobClusterLister{}

// CronJobClusterLister implements the batchv1beta1listers.CronJobLister interface.
type CronJobClusterLister struct {
	indexer cache.Indexer
}

// NewCronJobClusterLister returns a new CronJobClusterLister.
func NewCronJobClusterLister(indexer cache.Indexer) batchv1beta1listers.CronJobLister {
	return &CronJobClusterLister{indexer: indexer}
}

// List lists all batchv1beta1.CronJob in the indexer.
func (s CronJobClusterLister) List(selector labels.Selector) (ret []*batchv1beta1.CronJob, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*batchv1beta1.CronJob))
	})
	return ret, err
}

// CronJobs returns an object that can list and get batchv1beta1.CronJob.
func (s CronJobClusterLister) CronJobs(namespace string) batchv1beta1listers.CronJobNamespaceLister {
	panic("Calling 'CronJobs' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get batchv1beta1.CronJob.

func (s CronJobClusterLister) Cluster(cluster logicalcluster.Name) batchv1beta1listers.CronJobLister {
	return &CronJobLister{indexer: s.indexer, cluster: cluster}
}

// CronJobLister implements the batchv1beta1listers.CronJobLister interface.
type CronJobLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all batchv1beta1.CronJob in the indexer.
func (s CronJobLister) List(selector labels.Selector) (ret []*batchv1beta1.CronJob, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*batchv1beta1.CronJob)
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

// CronJobs returns an object that can list and get batchv1beta1.CronJob.
func (s CronJobLister) CronJobs(namespace string) batchv1beta1listers.CronJobNamespaceLister {
	return &CronJobNamespaceLister{indexer: s.indexer, cluster: s.cluster, namespace: namespace}
}

// CronJobNamespaceLister implements the batchv1beta1listers.CronJobNamespaceLister interface.
type CronJobNamespaceLister struct {
	indexer   cache.Indexer
	cluster   logicalcluster.Name
	namespace string
}

// List lists all batchv1beta1.CronJob in the indexer for a given namespace.
func (s CronJobNamespaceLister) List(selector labels.Selector) (ret []*batchv1beta1.CronJob, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterAndNamespaceIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*batchv1beta1.CronJob)
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

// Get retrieves the batchv1beta1.CronJob from the indexer for a given namespace and name.
func (s CronJobNamespaceLister) Get(name string) (*batchv1beta1.CronJob, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), s.namespace, name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(batchv1beta1.Resource("CronJob"), name)
	}
	return obj.(*batchv1beta1.CronJob), nil
}
