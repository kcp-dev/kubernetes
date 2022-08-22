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
	"github.com/kcp-dev/logicalcluster/v2"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	storagev1listers "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
)

var _ storagev1listers.VolumeAttachmentLister = &VolumeAttachmentClusterLister{}

// VolumeAttachmentClusterLister implements the storagev1listers.VolumeAttachmentLister interface.
type VolumeAttachmentClusterLister struct {
	indexer cache.Indexer
}

// NewVolumeAttachmentClusterLister returns a new VolumeAttachmentClusterLister.
func NewVolumeAttachmentClusterLister(indexer cache.Indexer) storagev1listers.VolumeAttachmentLister {
	return &VolumeAttachmentClusterLister{indexer: indexer}
}

// List lists all storagev1.VolumeAttachment in the indexer.
func (s VolumeAttachmentClusterLister) List(selector labels.Selector) (ret []*storagev1.VolumeAttachment, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*storagev1.VolumeAttachment))
	})
	return ret, err
}

// Get retrieves the storagev1.VolumeAttachment from the indexer for a given name.
func (s VolumeAttachmentClusterLister) Get(name string) (*storagev1.VolumeAttachment, error) {
	panic("Calling 'Get' is not supported before scoping lister to a workspace")
}

// Cluster returns an object that can list and get storagev1.VolumeAttachment.

func (s VolumeAttachmentClusterLister) Cluster(cluster logicalcluster.Name) storagev1listers.VolumeAttachmentLister {
	return &VolumeAttachmentLister{indexer: s.indexer, cluster: cluster}
}

// VolumeAttachmentLister implements the storagev1listers.VolumeAttachmentLister interface.
type VolumeAttachmentLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all storagev1.VolumeAttachment in the indexer.
func (s VolumeAttachmentLister) List(selector labels.Selector) (ret []*storagev1.VolumeAttachment, err error) {
	selectAll := selector == nil || selector.Empty()

	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", "")
	list, err := s.indexer.ByIndex(apimachinerycache.ClusterIndexName, key)
	if err != nil {
		return nil, err
	}

	for i := range list {
		obj := list[i].(*storagev1.VolumeAttachment)
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

// Get retrieves the storagev1.VolumeAttachment from the indexer for a given name.
func (s VolumeAttachmentLister) Get(name string) (*storagev1.VolumeAttachment, error) {
	key := apimachinerycache.ToClusterAwareKey(s.cluster.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(storagev1.Resource("VolumeAttachment"), name)
	}
	return obj.(*storagev1.VolumeAttachment), nil
}
