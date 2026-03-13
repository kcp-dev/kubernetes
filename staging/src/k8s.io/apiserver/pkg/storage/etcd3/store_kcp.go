/*
Copyright 2022 The KCP Authors.

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

package etcd3

import (
	"strings"

	"github.com/kcp-dev/logicalcluster/v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
)

// adjustKeyForPrefix returns storage path key stripped of its prefix, with corrections that
// account for wildcards and partial metadata if they are part of the request. We need that
// so the resulting key has a stable form, from which the shard and cluster name can be parsed.
//
// The key is in the following format:
//
//	<Storage prefix> / <Group> / <Resource> / [ <Identity or "customresources"> / ] [ <Shard> / ] <Cluster> / [ <Namespace> / ] <Name>
//
// The prefix is variable, and may include segments ranging from:
//
//	<Storage prefix> / <Group> / <Resource> /
//
// And up to:
//
//	<Storage prefix> / <Group> / <Resource> / [ <Identity or "customresources"> / ] [ <Shard> / ] <Cluster> /
//
// The exact prefix length depends on how inclusive the request is:
//
//   - Partial metadata requests accept any key with matching group-resource, regardless of its identity or
//     CustomResource origin. This makes the prefix stop right after .../<Resource>/. Only applies if CRDRequest=true.
//   - Shard wildcard requests accept any key with matching fully-qualified resource,
//     which makes the prefix stop right after .../<Identity or "customresources">/ in case of CRDRequest=true,
//     or right after .../<Resource>/ in case of CRDRequest=false.
//   - Cluster wildcard requests accept any key with matching fully-qualified resource, which makes the prefix
//     stop right .../<Shard>/, or .../<Identity or "customresources">/, or .../<Resource>/, depending on which
//     of those is part of the request.
//
// This function checks all these cases, and returns the key in format:
//
//   - If the shard in request is a wildcard: <Shard> / <Cluster> / [ <Namespace> / ] <Name>
//   - If the shard in request is empty or concrete:    <Cluster> / [ <Namespace> / ] <Name>
func adjustKeyForPrefix(prefix, key string, crdRequest bool, cluster *genericapirequest.Cluster, shard genericapirequest.Shard) string {
	popSegment := func(s string) string {
		segmentStart := strings.IndexByte(s, '/')
		if segmentStart < 0 {
			return s
		}
		if segmentStart == len(s) {
			return s
		}
		return s[segmentStart+1:]
	}

	keyWithoutPrefix := strings.TrimPrefix(key, prefix)

	// This is what keyWithoutPrefix can now contain, if:
	//
	//   [ <Identity or "customresources"> / ] [ <Shard> / ] <Cluster> / [ <Namespace> / ] <Name>
	//   ^                                     ^             ^           ^----------------------^
	//   |                                     |             |                    |             |
	//   |                                     |             |          (4) !cluster.Wildcard   |
	//   |                                     |             +----------------------------------+
	//   |                                     |                   |                            |
	//   |                                     |            (3) shard.Empty()                   |
	//   |                                     +------------------------------------------------+
	//   |                                                 |                                    |
	//   |                                        (2) !shard.Empty()                            |
	//   +--------------------------------------------------------------------------------------+
	//                  |
	//          (1) cluster.PartialMetadataRequest (assumes crdRequest)

	if !crdRequest || !cluster.PartialMetadataRequest {
		// The prefix already includes <Identity or "customresources"> segment,
		// or this is not a CRD request, so keyWithoutPrefix is clean.
		return keyWithoutPrefix
	}

	// With partial metadata, the prefix ends BEFORE the identity segment.
	keyWithoutIdentity := popSegment(keyWithoutPrefix)

	if shard.Empty() || shard.Wildcard() {
		// Shard needs to be part of the key if it is wildcard.
		return keyWithoutIdentity
	}

	// We are scoped to a shard, but the <Shard> segment wasn't part of the prefix
	// because of partial metadata ending it early, so we need to strip it from the key now.
	keyWithoutShard := popSegment(keyWithoutIdentity)

	return keyWithoutShard
}

// adjustClusterNameIfWildcard determines the logical cluster name. If this is not a cluster-wildcard list/watch request,
// the cluster name is returned unmodified. Otherwise, the cluster name is extracted from the storage key.
func adjustClusterNameIfWildcard(shard genericapirequest.Shard, cluster *genericapirequest.Cluster, crdRequest bool, keyPrefix, key string) logicalcluster.Name {
	if !cluster.Wildcard {
		return cluster.Name
	}

	keyWithoutPrefix := adjustKeyForPrefix(keyPrefix, key, crdRequest, cluster, shard)
	parts := strings.SplitN(keyWithoutPrefix, "/", 3)

	extract := func(minLen, i int) logicalcluster.Name {
		if len(parts) < minLen {
			klog.Warningf("shard=%s cluster=%v invalid key=%s had %d parts, wanted %d", shard, cluster, keyWithoutPrefix, len(parts), minLen)
			return ""
		}
		return logicalcluster.Name(parts[i])
	}

	if !shard.Wildcard() {
		// Shard is either empty or concrete. In both cases, adjustKeyForPrefix
		// has already stripped it (along with <Identity> segment, if applicable).
		//
		// The remaining key is in format:
		//   <Cluster> / <Remainder...>
		return extract(2, 0)
	}
	// Shard is wildcard, and still present in the key.
	//
	// The remaining key is in format:
	//   <Shard> / <Cluster> / <Remainder...>
	return extract(3, 1)
}

// adjustShardNameIfWildcard determines a shard name. If this is not a shard-wildcard request,
// the shard name is returned unmodified. Otherwise, the shard name is extracted from the storage key.
func adjustShardNameIfWildcard(shard genericapirequest.Shard, cluster *genericapirequest.Cluster, crdRequest bool, keyPrefix, key string) genericapirequest.Shard {
	if !shard.Empty() && !shard.Wildcard() {
		return shard
	}

	if !shard.Wildcard() {
		// no-op: we can only assign shard names
		// to a request that explicitly asked for it
		return ""
	}

	keyWithoutPrefix := adjustKeyForPrefix(keyPrefix, key, crdRequest, cluster, shard)

	// The remaining key is in format:
	//   <Shard> / <Cluster> / <Remainder...>
	parts := strings.SplitN(keyWithoutPrefix, "/", 3)
	if len(parts) < 3 {
		klog.Warningf("unable to extract a shard name, invalid key=%s had %d parts, wanted %d", keyWithoutPrefix, len(parts), 3)
		return ""
	}
	return genericapirequest.Shard(parts[0])
}

// annotateDecodedObjectWith applies clusterName and shardName to an object.
// This is necessary because we don't store the cluster name and the shard name in the objects in storage.
// Instead, they are derived from the storage key, and then applied after retrieving the object from storage.
func annotateDecodedObjectWith(obj interface{}, clusterName logicalcluster.Name, shardName genericapirequest.Shard) {
	var s nameSetter

	switch t := obj.(type) {
	case metav1.ObjectMetaAccessor:
		s = t.GetObjectMeta()
	case nameSetter:
		s = t
	default:
		klog.Warningf("Could not set ClusterName %s, ShardName %s on object: %T", clusterName, shardName, obj)
		return
	}

	annotations := s.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[logicalcluster.AnnotationKey] = clusterName.String()
	if !shardName.Empty() {
		annotations[genericapirequest.ShardAnnotationKey] = shardName.String()
	}
	s.SetAnnotations(annotations)
}

type nameSetter interface {
	GetAnnotations() map[string]string
	SetAnnotations(a map[string]string)
}
