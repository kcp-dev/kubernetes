/*
Copyright 2026 The kcp Authors.

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

package garbagecollector

import (
	"context"

	kcpcache "github.com/kcp-dev/apimachinery/v2/pkg/cache"
	kcpkubernetesclientset "github.com/kcp-dev/client-go/kubernetes"
	"github.com/kcp-dev/logicalcluster/v3"
	v1 "k8s.io/api/core/v1"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

type clusterContextKey struct{}

func WithCluster(ctx context.Context, cluster logicalcluster.Name) context.Context {
	return context.WithValue(ctx, clusterContextKey{}, cluster)
}

func ClusterFrom(ctx context.Context) logicalcluster.Name {
	v, _ := ctx.Value(clusterContextKey{}).(logicalcluster.Name)
	return v
}

func contextForCluster(ref objectReference) context.Context {
	return WithCluster(context.TODO(), ref.Cluster)
}

var _ record.EventSink = (*clusterEventSinkImpl)(nil)

// clusterEventSinkImpl allows the non-cluster aware garbage collector
// to emit cluster aware events.
//
// The core problem is that the broadcaster writes to sinks and events
// are written to recorders build from broadcasters:
// eventf(ref) -> recorder -> broadcaster -> sink
//
// To make this chain cluster aware either the object reference would
// have to be cluster aware - which could produce side effects because
// no other code is expected object references to be cluster-aware. It
// could also confuse other tooling if starting to rely on this being
// cluster aware.
//
// Instead the producer is patched to put the cluster name into the name
// on the object reference and this sink then extracts the name from
// there.
// This works because the recorder and broadcaster are accepting the
// object reference as-is and place it in the InvolvedObject property of
// the event to be emitted, allowing the sink to extract the cluster
// name and fix the name.
type clusterEventSinkImpl struct {
	client *kcpkubernetesclientset.ClusterClientset
}

func (sink *clusterEventSinkImpl) getClient(event *v1.Event) v1core.EventInterface {
	cluster, _, name, err := kcpcache.SplitMetaClusterNamespaceKey(event.InvolvedObject.Name)
	if err != nil {
		panic("error splitting name into cluster and name in garbage collector")
	}

	event.InvolvedObject.Name = name
	return sink.client.Cluster(cluster.Path()).CoreV1().Events("")
}

func (sink *clusterEventSinkImpl) Create(event *v1.Event) (*v1.Event, error) {
	return sink.getClient(event).CreateWithEventNamespace(event)
}

func (sink *clusterEventSinkImpl) Update(event *v1.Event) (*v1.Event, error) {
	return sink.getClient(event).UpdateWithEventNamespace(event)
}

func (sink *clusterEventSinkImpl) Patch(event *v1.Event, data []byte) (*v1.Event, error) {
	return sink.getClient(event).PatchWithEventNamespace(event, data)
}
