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

package etcd3

import (
	"context"
	"errors"
	"reflect"

	"go.etcd.io/etcd/api/v3/mvccpb"
	etcdrpc "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apiserver/pkg/storage/generic"
)

func transform(resp *clientv3.GetResponse) *generic.Response {
	if resp == nil {
		return nil
	}
	var kvs []*generic.KeyValue
	for i := range resp.Kvs {
		kvs = append(kvs, &generic.KeyValue{
			Key:         resp.Kvs[i].Key,
			Value:       resp.Kvs[i].Value,
			ModRevision: resp.Kvs[i].ModRevision,
		})
		resp.Kvs[i] = nil // otherwise, we increase memory footprint while iterating
	}
	return &generic.Response{
		Header: &generic.ResponseHeader{Revision: resp.Header.Revision},
		Kvs:    kvs,
		Count:  resp.Count,
		More:   resp.More,
	}
}

func transformPut(resp *clientv3.PutResponse) *generic.Response {
	if resp == nil {
		return nil
	}
	return &generic.Response{
		Header: &generic.ResponseHeader{Revision: resp.Header.Revision},
	}
}

func NewClient(c *clientv3.Client, config LeaseManagerConfig) *client {
	return &client{
		Client: c,
		leaseManager: newDefaultLeaseManager(c, config),
	}
}

type client struct {
	*clientv3.Client
	leaseManager *leaseManager
}

func (c *client) Get(ctx context.Context, key string) (response *generic.Response, err error) {
	getResp, err := c.KV.Get(ctx, key)
	return transform(getResp), err
}

func (c *client) Create(ctx context.Context, key string, value []byte, ttl uint64) (succeeded bool, response *generic.Response, err error) {
	opts, err := c.ttlOpts(ctx, int64(ttl))
	if err != nil {
		return false, nil, err
	}

	txnResp, err := c.KV.Txn(ctx).If(
		notFound(key),
	).Then(
		clientv3.OpPut(key, string(value), opts...),
	).Commit()
	if err != nil {
		return false, nil, err
	}

	var resp *generic.Response
	if txnResp.Succeeded {
		resp = transformPut((*clientv3.PutResponse)(txnResp.Responses[0].GetResponsePut()))
	}
	return txnResp.Succeeded, resp, err
}

func (c *client) ConditionalDelete(ctx context.Context, key string, previousRevision int64) (succeeded bool, response *generic.Response, err error) {
	txnResp, err := c.KV.Txn(ctx).If(
		clientv3.Compare(clientv3.ModRevision(key), "=", previousRevision),
	).Then(
		clientv3.OpDelete(key),
	).Else(
		clientv3.OpGet(key),
	).Commit()
	if err != nil {
		return false, nil, err
	}

	var resp *generic.Response
	if !txnResp.Succeeded {
		resp = transform((*clientv3.GetResponse)(txnResp.Responses[0].GetResponseRange()))
	}
	return txnResp.Succeeded, resp, err
}

func (c *client) ConditionalUpdate(ctx context.Context, key string, value []byte, previousRevision int64, ttl uint64) (succeeded bool, response *generic.Response, err error) {
	opts, err := c.ttlOpts(ctx, int64(ttl))
	if err != nil {
		return false, nil, err
	}

	txnResp, err := c.KV.Txn(ctx).If(
		clientv3.Compare(clientv3.ModRevision(key), "=", previousRevision),
	).Then(
		clientv3.OpPut(key, string(value), opts...),
	).Else(
		clientv3.OpGet(key),
	).Commit()
	if err != nil {
		return false, nil, err
	}

	var resp *generic.Response
	if !txnResp.Succeeded {
		resp = transform((*clientv3.GetResponse)(txnResp.Responses[0].GetResponseRange()))
	} else {
		resp = transformPut((*clientv3.PutResponse)(txnResp.Responses[0].GetResponsePut()))
	}
	return txnResp.Succeeded, resp, err
}

func (c *client) Count(ctx context.Context, key string) (count int64, err error) {
	getResp, err := c.KV.Get(ctx, key, clientv3.WithRange(clientv3.GetPrefixRangeEnd(key)), clientv3.WithCountOnly())
	if err != nil {
		return 0, err
	}
	return getResp.Count, nil
}

func (c *client) List(ctx context.Context, key, prefix string, recursive, paging bool, limit, revision int64) (response *generic.Response, err error) {
	// set the appropriate clientv3 options to filter the returned data set
	options := make([]clientv3.OpOption, 0, 4)
	if limit > 0 {
		options = append(options, clientv3.WithLimit(limit))
	}
	if revision != 0 {
		options = append(options, clientv3.WithRev(revision))
	}
	if recursive {
		keyPrefix := key
		if paging {
			keyPrefix = prefix
		}
		rangeEnd := clientv3.GetPrefixRangeEnd(keyPrefix)
		options = append(options, clientv3.WithRange(rangeEnd))
	}

	//startTime := time.Now()
	getResp, err := c.KV.Get(ctx, key, options...)
	if recursive {
		//metrics.RecordEtcdRequestLatency("list", getTypeName(nil), startTime)
	} else {
		//metrics.RecordEtcdRequestLatency("get", getTypeName(nil), startTime)
	}

	return transform(getResp), transformError(err)
}

func transformWatch(wres clientv3.WatchResponse) *generic.WatchResponse {
	var events []*generic.WatchEvent
	for i := range wres.Events {
		evt := &generic.WatchEvent{}
		switch {
		case wres.Events[i].IsCreate():
			evt.Type = generic.EventTypeCreate
		case wres.Events[i].IsModify():
			evt.Type = generic.EventTypeModify
		case wres.Events[i].Type == mvccpb.DELETE:
			evt.Type = generic.EventTypeDelete
		default:
			evt.Type = generic.EventTypeUnknown
		}
		if wres.Events[i].Kv != nil {
			evt.Kv = &generic.KeyValue{
				Key:         wres.Events[i].Kv.Key,
				Value:       wres.Events[i].Kv.Value,
				ModRevision: wres.Events[i].Kv.ModRevision,
			}
		}
		if wres.Events[i].PrevKv != nil {
			evt.PrevKv = &generic.KeyValue{
				Key:         wres.Events[i].PrevKv.Key,
				Value:       wres.Events[i].PrevKv.Value,
				ModRevision: wres.Events[i].PrevKv.ModRevision,
			}
		}
		events = append(events, evt)
	}
	return &generic.WatchResponse{
		Header:         &generic.ResponseHeader{Revision: wres.Header.Revision},
		ProgressNotify: wres.IsProgressNotify(),
		Events:         events,
		Error:          transformError(wres.Err()),
	}
}

func transformError(err error) error {
	if errors.Is(err, etcdrpc.ErrCompacted) {
		return generic.ErrCompacted
	}
	return err
}

func (c *client) Watch(ctx context.Context, key string, recursive, progressNotify bool, revision int64) <-chan *generic.WatchResponse {
	opts := []clientv3.OpOption{clientv3.WithRev(revision), clientv3.WithPrevKV()}
	if recursive {
		opts = append(opts, clientv3.WithPrefix())
	}
	if progressNotify {
		opts = append(opts, clientv3.WithProgressNotify())
	}
	out := make(chan *generic.WatchResponse)
	go func() {
		for wres := range c.Client.Watch(ctx, key, opts...) {
			out <- transformWatch(wres)
		}
	}()
	return out
}

// ttlOpts returns client options based on given ttl.
// ttl: if ttl is non-zero, it will attach the key to a lease with ttl of roughly the same length
func (c *client) ttlOpts(ctx context.Context, ttl int64) ([]clientv3.OpOption, error) {
	if ttl == 0 {
		return nil, nil
	}
	id, err := c.leaseManager.GetLease(ctx, ttl)
	if err != nil {
		return nil, err
	}
	return []clientv3.OpOption{clientv3.WithLease(id)}, nil
}

var _ generic.Client = &client{}

func notFound(key string) clientv3.Cmp {
	return clientv3.Compare(clientv3.ModRevision(key), "=", 0)
}

// getTypeName returns type name of an object for reporting purposes.
func getTypeName(obj interface{}) string {
	return reflect.TypeOf(obj).String()
}
