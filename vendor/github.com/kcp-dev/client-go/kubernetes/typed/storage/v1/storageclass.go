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
	"context"

	kcpclient "github.com/kcp-dev/apimachinery/v2/pkg/client"
	"github.com/kcp-dev/logicalcluster/v3"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	storagev1client "k8s.io/client-go/kubernetes/typed/storage/v1"
)

// StorageClassesClusterGetter has a method to return a StorageClassClusterInterface.
// A group's cluster client should implement this interface.
type StorageClassesClusterGetter interface {
	StorageClasses() StorageClassClusterInterface
}

// StorageClassClusterInterface can operate on StorageClasses across all clusters,
// or scope down to one cluster and return a storagev1client.StorageClassInterface.
type StorageClassClusterInterface interface {
	Cluster(logicalcluster.Path) storagev1client.StorageClassInterface
	List(ctx context.Context, opts metav1.ListOptions) (*storagev1.StorageClassList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type storageClassesClusterInterface struct {
	clientCache kcpclient.Cache[*storagev1client.StorageV1Client]
}

// Cluster scopes the client down to a particular cluster.
func (c *storageClassesClusterInterface) Cluster(clusterPath logicalcluster.Path) storagev1client.StorageClassInterface {
	if clusterPath == logicalcluster.Wildcard {
		panic("A specific cluster must be provided when scoping, not the wildcard.")
	}

	return c.clientCache.ClusterOrDie(clusterPath).StorageClasses()
}

// List returns the entire collection of all StorageClasses across all clusters.
func (c *storageClassesClusterInterface) List(ctx context.Context, opts metav1.ListOptions) (*storagev1.StorageClassList, error) {
	return c.clientCache.ClusterOrDie(logicalcluster.Wildcard).StorageClasses().List(ctx, opts)
}

// Watch begins to watch all StorageClasses across all clusters.
func (c *storageClassesClusterInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.clientCache.ClusterOrDie(logicalcluster.Wildcard).StorageClasses().Watch(ctx, opts)
}
