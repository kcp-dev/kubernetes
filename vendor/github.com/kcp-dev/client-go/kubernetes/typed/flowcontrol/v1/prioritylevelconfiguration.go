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

	flowcontrolv1 "k8s.io/api/flowcontrol/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	flowcontrolv1client "k8s.io/client-go/kubernetes/typed/flowcontrol/v1"
)

// PriorityLevelConfigurationsClusterGetter has a method to return a PriorityLevelConfigurationClusterInterface.
// A group's cluster client should implement this interface.
type PriorityLevelConfigurationsClusterGetter interface {
	PriorityLevelConfigurations() PriorityLevelConfigurationClusterInterface
}

// PriorityLevelConfigurationClusterInterface can operate on PriorityLevelConfigurations across all clusters,
// or scope down to one cluster and return a flowcontrolv1client.PriorityLevelConfigurationInterface.
type PriorityLevelConfigurationClusterInterface interface {
	Cluster(logicalcluster.Path) flowcontrolv1client.PriorityLevelConfigurationInterface
	List(ctx context.Context, opts metav1.ListOptions) (*flowcontrolv1.PriorityLevelConfigurationList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type priorityLevelConfigurationsClusterInterface struct {
	clientCache kcpclient.Cache[*flowcontrolv1client.FlowcontrolV1Client]
}

// Cluster scopes the client down to a particular cluster.
func (c *priorityLevelConfigurationsClusterInterface) Cluster(clusterPath logicalcluster.Path) flowcontrolv1client.PriorityLevelConfigurationInterface {
	if clusterPath == logicalcluster.Wildcard {
		panic("A specific cluster must be provided when scoping, not the wildcard.")
	}

	return c.clientCache.ClusterOrDie(clusterPath).PriorityLevelConfigurations()
}

// List returns the entire collection of all PriorityLevelConfigurations across all clusters.
func (c *priorityLevelConfigurationsClusterInterface) List(ctx context.Context, opts metav1.ListOptions) (*flowcontrolv1.PriorityLevelConfigurationList, error) {
	return c.clientCache.ClusterOrDie(logicalcluster.Wildcard).PriorityLevelConfigurations().List(ctx, opts)
}

// Watch begins to watch all PriorityLevelConfigurations across all clusters.
func (c *priorityLevelConfigurationsClusterInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.clientCache.ClusterOrDie(logicalcluster.Wildcard).PriorityLevelConfigurations().Watch(ctx, opts)
}
