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
	"sync/atomic"
	"testing"

	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd3/testserver"
	"k8s.io/apiserver/pkg/storage/value"
)

func TestBoostrapper(leaseConfig *LeaseManagerConfig) storage.TestBootstrapper {
	// our default, for brevity:
	if leaseConfig == nil {
		leaseConfig = &LeaseManagerConfig{
			ReuseDurationSeconds: 1,
			MaxObjectCount:       defaultLeaseMaxObjectCount,
		}
	}
	return &etcdTestBootstrapper{leaseConfig: *leaseConfig}
}

type etcdTestBootstrapper struct {
	leaseConfig LeaseManagerConfig
}

func (e *etcdTestBootstrapper) InterfaceForClient(t *testing.T, client storage.InternalTestClient, codec runtime.Codec, newFunc func() runtime.Object, prefix string, groupResource schema.GroupResource, transformer value.Transformer, pagingEnabled bool) storage.InternalTestInterface {
	etcdClient, ok := client.(*etcdTestClient)
	if !ok {
		t.Fatalf("got a %T, not etcd test client", client)
	}
	store := newStore(etcdClient.client, codec, newFunc, prefix, groupResource, transformer, pagingEnabled, e.leaseConfig)
	return &etcdTestInterface{store: store}
}

func (e *etcdTestBootstrapper) Setup(t *testing.T, codec runtime.Codec, newFunc func() runtime.Object, prefix string, groupResource schema.GroupResource, transformer value.Transformer, pagingEnabled bool) (context.Context, storage.InternalTestInterface, storage.InternalTestClient) {
	client := testserver.RunEtcd(t, nil)
	kv := &recordingClient{KV: client.KV}
	client.KV = kv
	store := newStore(client, codec, newFunc, prefix, groupResource, transformer, pagingEnabled, e.leaseConfig)
	ctx := context.Background()
	clusterName := "admin"
	ctx = request.WithCluster(ctx, request.Cluster{Name: clusterName})
	ctx = request.WithRequestInfo(ctx, &request.RequestInfo{
		APIGroup:   "",
		APIVersion: "v1",
		Namespace:  "ns",
		Resource:   "pods",
		Name:       "foo",
	})
	return ctx, &etcdTestInterface{store: store}, &etcdTestClient{recordingClient: kv, client: client}
}

type etcdTestClient struct {
	*recordingClient

	client *clientv3.Client
}

type recordingClient struct {
	reads uint64
	clientv3.KV
}

func (r *recordingClient) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	atomic.AddUint64(&r.reads, 1)
	return r.KV.Get(ctx, key, opts...)
}

func (e *etcdTestClient) Reads() uint64 {
	return atomic.LoadUint64(&e.reads)
}

func (e *etcdTestClient) ResetReads() {
	atomic.StoreUint64(&e.reads, 0)
}

func (e *etcdTestClient) RawPut(ctx context.Context, key, data string) (int64, error) {
	resp, err := e.Put(ctx, key, data)
	return resp.Header.Revision, err
}

func (e *etcdTestClient) RawGet(ctx context.Context, key string) ([]byte, bool, error) {
	getResp, err := e.KV.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}
	if len(getResp.Kvs) == 0 {
		return nil, true, nil
	}
	return getResp.Kvs[0].Value, false, nil
}

func (e *etcdTestClient) RawCompact(ctx context.Context, revision int64) error {
	_, err := e.KV.Compact(ctx, revision, clientv3.WithCompactPhysical())
	return err
}

type etcdTestInterface struct {
	*store
}

func (e *etcdTestInterface) PathPrefix() string {
	return e.pathPrefix
}

func (e *etcdTestInterface) UpdateTransformer(f func(transformer value.Transformer) value.Transformer) {
	e.transformer = f(e.transformer)
}

func (e *etcdTestInterface) Decode(raw []byte) (runtime.Object, error) {
	return runtime.Decode(e.codec, raw)
}
