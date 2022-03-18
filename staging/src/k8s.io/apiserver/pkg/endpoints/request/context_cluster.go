/*
Copyright 2014 The Kubernetes Authors.

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

package request

import (
	"context"
	"errors"
	"fmt"

	"github.com/kcp-dev/apimachinery/pkg/logicalcluster"
)

type clusterKey int

const (
	// clusterKey is the context key for the request namespace.
	clusterContextKey clusterKey = iota
)

type Cluster struct {
	// Name is the name of the cluster.
	Name logicalcluster.LogicalCluster

	// HACK: only for testing wildcard semantics
	// If true the query applies to all clusters
	Wildcard bool
}

// WithCluster returns a context that describes the nested cluster context
func WithCluster(parent context.Context, cluster Cluster) context.Context {
	return context.WithValue(parent, clusterContextKey, cluster)
}

// ClusterFrom returns the value of the cluster key on the ctx, or nil if there
// is no cluster key.
func ClusterFrom(ctx context.Context) *Cluster {
	cluster, ok := ctx.Value(clusterContextKey).(Cluster)
	if !ok {
		return nil
	}
	return &cluster
}

func buildClusterError(message string, ctx context.Context) error {
	if ri, ok := RequestInfoFrom(ctx); ok {
		message = message + fmt.Sprintf(" - RequestInfo: %#v", ri)
	}
	return errors.New(message)
}

// ValidClusterFrom returns the value of the cluster key on the ctx.
// If there's no cluster key, or if the cluster name is empty
// and it's not a wildcard context, then return an error.
func ValidClusterFrom(ctx context.Context) (*Cluster, error) {
	cluster := ClusterFrom(ctx)
	if cluster == nil {
		return nil, buildClusterError("no cluster in the request context", ctx)
	}
	if cluster.Name.Empty() && !cluster.Wildcard {
		return nil, buildClusterError("cluster name is empty in the request context", ctx)
	}
	return cluster, nil
}

// ClusterNameFrom returns the cluster name from the value of the cluster
// key on the ctx.
// If the cluster name is empty, then return an error.
func ClusterNameFrom(ctx context.Context) (logicalcluster.LogicalCluster, error) {
	cluster, err := ValidClusterFrom(ctx)
	if err != nil {
		return logicalcluster.LogicalCluster{}, err
	}
	if cluster.Name.Empty() {
		return logicalcluster.LogicalCluster{}, buildClusterError("cluster name is empty in the request context", ctx)
	}
	return cluster.Name, nil
}
