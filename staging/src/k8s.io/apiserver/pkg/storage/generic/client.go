/*
Copyright 2022 The Kubernetes Authors.

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

package generic

import (
	"context"
)

// Response holds data coming from a storage backend RPC.
type Response struct {
	// Header contains metadata for the transaction.
	Header *ResponseHeader
	// Kvs contains any key-value pairs the RPC returned.
	Kvs []*KeyValue

	// Count contains the number of key-value pairs that matched the RPC, if requested.
	Count int64
	// More determines if the current Kvs contain all of the matching key-value pairs or not.
	More bool
}

// ResponseHeader holds metadata for a storage transaction.
type ResponseHeader struct {
	// Revision is the logical clock at which the storage transaction was committed.
	Revision int64
}

// KeyValue represents a key-value pair in the underlying store.
type KeyValue struct {
	Key   []byte
	Value []byte

	// ModRevision is the logical clock at which this key-value pair was last updated.
	ModRevision int64
}

// WatchResponse holds metadata and events from a Watch RPC stream.
type WatchResponse struct {
	// Header contains metadata for the stream.
	Header         *ResponseHeader
	// ProgressNotify determines if this response is a progress notification.
	ProgressNotify bool

	// Events are the watch events.
	Events []*WatchEvent
	// Error holds any errors encountered.
	Error  error
}

// WatchEvent represents a mutation to a key-value pair in the underlying storage.
type WatchEvent struct {
	// Type is the type of event that occurred.
	Type   EventType
	// Kv is the current value of the key-value pair, if any.
	Kv     *KeyValue
	// PrevKv is the previous value of the key-value pair, if any.
	PrevKv *KeyValue
}

// IsCreate determines if the event is a key-value pair creation.
func (e *WatchEvent) IsCreate() bool {
	return e.Type == EventTypeCreate
}

// EventType denotes the type of event that occurred.
type EventType int8

const (
	EventTypeCreate EventType = iota
	EventTypeDelete
	EventTypeModify
	EventTypeUnknown
)

func (t EventType) String() string {
	switch t {
	case EventTypeCreate:
		return "create"
	case EventTypeModify:
		return "modify"
	case EventTypeDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Client is a "generic" interface for a key-value store that exposes the MVCC logical timestamps
// in order to allow clients to manually enforce opportunistic locking.
type Client interface {
	// Get returns the value of key as of the call-time of this RPC.
	// When the key does not exist, no error should be returned; instead,
	// the response should contain no key-value pairs.
	Get(ctx context.Context, key string) (response *Response, err error)
	// Create stores the key-value pair with the provided ttl, if any.
	// If the key already exists, no error should be returned; instead,
	// `succeeded` should be set to false.
	Create(ctx context.Context, key string, value []byte, ttl uint64) (succeeded bool, response *Response, err error)
	// ConditionalDelete deletes the key-value pair from the underlying
	// storage if and only if the key-value pair has not changed from the
	// provided previous revision. If any changes have occurred; `succeeded`
	// should be set to false and `response` should contain the updated key-
	// value pair.
	ConditionalDelete(ctx context.Context, key string, previousRevision int64) (succeeded bool, response *Response, err error)
	// ConditionalUpdate updates the key-value pair from the underlying
	// storage if and only if the key-value pair has not changed from the
	// provided previous revision. If any changes have occurred; `succeeded`
	// should be set to false and `response` should contain the updated key-
	// value pair.
	ConditionalUpdate(ctx context.Context, key string, value []byte, previousRevision int64, ttl uint64) (succeeded bool, response *Response, err error)
	// Count returns the count of objects with keys that have the provide key
	// as a prefix.
	Count(ctx context.Context, key string) (count int64, err error)
	// List returns matching key-value pairs with the following properties:
	// - if a limit is provided and the call is paginated,
	//   no more than limit key-value pairs are returned
	// - key-value pairs are returned at the given revision,
	//   if any is provided, or as of the call-time of the RPC
	//   otherwise
	// - key-value pairs are deemed to match the query if:
	//   - `prefix` is a prefix of the key
	//   - `key` is lexicographically smaller than the key, if the
	//     query is recursive, or `key` matches the key exactly, otherwise
	List(ctx context.Context, key, prefix string, recursive, paging bool, limit, revision int64) (response *Response, err error)
	// Watch sends events occurring after the revision for matching key-value
	// pairs to the returned channel, providing progress notification if requested.
	// Key-value pairs are deemed to match the query if:
	// - `key` is a prefix of the key, if the query is recursive, or
	// - `key` matches the key exactly, otherwise
	Watch(ctx context.Context, key string, recursive, progressNotify bool, revision int64) <-chan *WatchResponse
}

// TestClient is a strict superset of the generic Client above, exposing compaction.
type TestClient interface {
	Client
	// Compact causes all revisions <= revision to be removed.
	Compact(ctx context.Context, revision int64) error
}

// InternalTestClient is a strict superset of the generic TestClient above, exposing
// some utility methods to aid in testing a store.Interface with a given Client as
// the driver.
type InternalTestClient interface {
	TestClient
	ReadRecorder
}

// ReadRecorder knows how to record the number of reads that a client makes to the underlying storage.
type ReadRecorder interface {
	// Reads returns the number of reads issued to the underlying storage
	// since the last time that ResetReads() was called.
	Reads() uint64
	// ResetReads resets the counter of reads for this client.
	ResetReads()
}
