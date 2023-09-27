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

package registry

import (
	"context"

	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

// NoNamespaceKeyRootFunc is the default function for constructing storage
// paths to resource directories. In kcp, all storage keys are scoped by the
// logical cluster of the request, i.e. <prefix>/<clusterName>[/...], so that
// workspaces are isolated in etcd. Wildcard requests span all clusters and
// therefore stay at <prefix>.
func NoNamespaceKeyRootFunc(ctx context.Context, prefix string) string {
	key := prefix
	cluster, err := genericapirequest.ValidClusterFrom(ctx)
	if err != nil {
		klog.Errorf("invalid context cluster value: %v", err)
		return key
	}
	if !cluster.Wildcard {
		key += "/" + cluster.Name.String()
	}
	return key
}
