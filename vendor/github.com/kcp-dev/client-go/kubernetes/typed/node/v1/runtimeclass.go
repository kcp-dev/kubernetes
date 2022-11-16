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

package v1

import (
	"context"

	kcpclient "github.com/kcp-dev/apimachinery/pkg/client"
	"github.com/kcp-dev/logicalcluster/v2"

	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	nodev1client "k8s.io/client-go/kubernetes/typed/node/v1"
)

// RuntimeClassesClusterGetter has a method to return a RuntimeClassClusterInterface.
// A group's cluster client should implement this interface.
type RuntimeClassesClusterGetter interface {
	RuntimeClasses() RuntimeClassClusterInterface
}

// RuntimeClassClusterInterface can operate on RuntimeClasses across all clusters,
// or scope down to one cluster and return a nodev1client.RuntimeClassInterface.
type RuntimeClassClusterInterface interface {
	Cluster(logicalcluster.Name) nodev1client.RuntimeClassInterface
	List(ctx context.Context, opts metav1.ListOptions) (*nodev1.RuntimeClassList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type runtimeClassesClusterInterface struct {
	clientCache kcpclient.Cache[*nodev1client.NodeV1Client]
}

// Cluster scopes the client down to a particular cluster.
func (c *runtimeClassesClusterInterface) Cluster(name logicalcluster.Name) nodev1client.RuntimeClassInterface {
	if name == logicalcluster.Wildcard {
		panic("A specific cluster must be provided when scoping, not the wildcard.")
	}

	return c.clientCache.ClusterOrDie(name).RuntimeClasses()
}

// List returns the entire collection of all RuntimeClasses across all clusters.
func (c *runtimeClassesClusterInterface) List(ctx context.Context, opts metav1.ListOptions) (*nodev1.RuntimeClassList, error) {
	return c.clientCache.ClusterOrDie(logicalcluster.Wildcard).RuntimeClasses().List(ctx, opts)
}

// Watch begins to watch all RuntimeClasses across all clusters.
func (c *runtimeClassesClusterInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.clientCache.ClusterOrDie(logicalcluster.Wildcard).RuntimeClasses().Watch(ctx, opts)
}
