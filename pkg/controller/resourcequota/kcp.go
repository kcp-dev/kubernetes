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

package resourcequota

import (
	"context"
	"strconv"
	"strings"

	"github.com/kcp-dev/logicalcluster/v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	clusterNamespaceDelimiter = "|"

	kcpClusterScopedQuotaNamespace                 = "admin"
	kcpExperimentalClusterScopedQuotaAnnotationKey = "experimental.quota.kcp.io/cluster-scoped"
)

// EncodeNamespace produces the <cluster>|<namespace> form expected in metadata.namespace when the kubequota singleton is in use.
func EncodeNamespace(cluster logicalcluster.Name, namespace string) string {
	return cluster.String() + clusterNamespaceDelimiter + namespace
}

// ParseEncodedNamespace splits a <cluster>|<namespace> namespace produced by EncodeNamespace.
func ParseEncodedNamespace(encoded string) (logicalcluster.Name, string) {
	if i := strings.Index(encoded, clusterNamespaceDelimiter); i >= 0 {
		return logicalcluster.Name(encoded[:i]), encoded[i+1:]
	}
	return logicalcluster.Name(""), encoded
}

// IsEncodedNamespace reports whether ns has been encoded by EncodeNamespace.
func IsEncodedNamespace(ns string) bool {
	return strings.Contains(ns, clusterNamespaceDelimiter)
}

// namespaceToCheck returns the namespace that CalculateUsage should scope to
// for rq. For cluster-scoped quotas (annotated with
// experimental.quota.kcp.io/cluster-scoped=true in the "admin" namespace) it
// drops the namespace component but keeps the cluster prefix so the lister
// scopes to all objects in the workspace.
func namespaceToCheck(rq *v1.ResourceQuota) string {
	cluster, plainNs := ParseEncodedNamespace(rq.Namespace)
	clusterScoped, _ := strconv.ParseBool(rq.Annotations[kcpExperimentalClusterScopedQuotaAnnotationKey])
	if plainNs == kcpClusterScopedQuotaNamespace && clusterScoped {
		return EncodeNamespace(cluster, "")
	}
	return rq.Namespace
}

// ResyncMonitors allows kcp to directly sync monitors.
func (rq *Controller) ResyncMonitors(ctx context.Context, resources map[schema.GroupVersionResource]struct{}) error {
	return rq.resyncMonitors(ctx, resources)
}

// EnqueueAll re-enqueues every quota to recompute .status.used.
func (rq *Controller) EnqueueAll(ctx context.Context) {
	rq.enqueueAll(ctx)
}
