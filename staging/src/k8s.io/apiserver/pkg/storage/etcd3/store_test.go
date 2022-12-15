/*
Copyright 2016 The Kubernetes Authors.

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
	"go.etcd.io/etcd/server/v3/embed"
	"k8s.io/apiserver/pkg/storage/generic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/apis/example"
	"k8s.io/apiserver/pkg/storage/etcd3/testserver"
)

func newTestLeaseManagerConfig() LeaseManagerConfig {
	cfg := NewDefaultLeaseManagerConfig()
	// As 30s is the default timeout for testing in global configuration,
	// we cannot wait longer than that in a single time: change it to 1s
	// for testing purposes. See wait.ForeverTestTimeout
	cfg.ReuseDurationSeconds = 1
	return cfg
}

type internalTestClient struct {
	*client
	generic.ReadRecorder
}

var _ generic.InternalTestClient = &internalTestClient{}

func (c *internalTestClient) Compact(ctx context.Context, revision int64) error {
	_, err := c.client.KV.Compact(ctx, revision, clientv3.WithCompactPhysical())
	return err
}

type clientRecorder struct {
	reads uint64
	clientv3.KV
}

func (r *clientRecorder) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	atomic.AddUint64(&r.reads, 1)
	return r.KV.Get(ctx, key, opts...)
}

func (r *clientRecorder) ResetReads() {
	r.reads = 0
}

func (r *clientRecorder) Reads() uint64 {
	return r.reads
}

func newTestClientCfg(t *testing.T, cfg *embed.Config) *internalTestClient {
	etcdClient := testserver.RunEtcd(t, cfg)
	recorder := &clientRecorder{KV: etcdClient.KV}
	etcdClient.KV = recorder
	return &internalTestClient{
		client: &client{
			Client:       etcdClient,
			leaseManager: newDefaultLeaseManager(etcdClient, newTestLeaseManagerConfig()),
		},
		ReadRecorder: recorder,
	}
}

func newTestClient(t *testing.T) *internalTestClient {
	return newTestClientCfg(t, nil)
}

func TestCreate(t *testing.T) {
	generic.RunTestCreate(t, newTestClient(t))
}

func TestCreateWithTTL(t *testing.T) {
	generic.RunTestCreateWithTTL(t, newTestClient(t))
}

func TestCreateWithKeyExist(t *testing.T) {
	generic.RunTestCreateWithKeyExist(t, newTestClient(t))
}

func TestGet(t *testing.T) {
	generic.RunTestGet(t, newTestClient(t))
}

func TestUnconditionalDelete(t *testing.T) {
	generic.RunTestUnconditionalDelete(t, newTestClient(t))
}

func TestConditionalDelete(t *testing.T) {
	generic.RunTestConditionalDelete(t, newTestClient(t))
}

func TestDeleteWithSuggestion(t *testing.T) {
	generic.RunTestDeleteWithSuggestion(t, newTestClient(t))
}

func TestDeleteWithSuggestionAndConflict(t *testing.T) {
	generic.RunTestDeleteWithSuggestionAndConflict(t, newTestClient(t))
}

func TestDeleteWithSuggestionOfDeletedObject(t *testing.T) {
	generic.RunTestDeleteWithSuggestionOfDeletedObject(t, newTestClient(t))
}

func TestValidateDeletionWithSuggestion(t *testing.T) {
	generic.RunTestValidateDeletionWithSuggestion(t, newTestClient(t))
}

func TestPreconditionalDeleteWithSuggestion(t *testing.T) {
	generic.RunTestPreconditionalDeleteWithSuggestion(t, newTestClient(t))
}

func TestGetListNonRecursive(t *testing.T) {
	generic.RunTestGetListNonRecursive(t, newTestClient(t))
}

func TestGuaranteedUpdate(t *testing.T) {
	generic.RunTestGuaranteedUpdate(t, newTestClient(t))
}

func TestGuaranteedUpdateWithTTL(t *testing.T) {
	generic.RunTestGuaranteedUpdateWithTTL(t, newTestClient(t))
}

func TestGuaranteedUpdateChecksStoredData(t *testing.T) {
	generic.RunTestGuaranteedUpdateChecksStoredData(t, newTestClient(t))
}

func TestGuaranteedUpdateWithConflict(t *testing.T) {
	generic.RunTestGuaranteedUpdateWithConflict(t, newTestClient(t))
}

func TestGuaranteedUpdateWithSuggestionAndConflict(t *testing.T) {
	generic.RunTestGuaranteedUpdateWithSuggestionAndConflict(t, newTestClient(t))
}

func TestTransformationFailure(t *testing.T) {
	generic.RunTestTransformationFailure(t, newTestClient(t))
}

func TestList(t *testing.T) {
	generic.RunTestList(t, newTestClient(t))
}

func TestListContinuation(t *testing.T) {
	generic.RunTestListContinuation(t, newTestClient(t))
}

func TestListPaginationRareObject(t *testing.T) {
	generic.RunTestListPaginationRareObject(t, newTestClient(t))
}

func TestListContinuationWithFilter(t *testing.T) {
	generic.RunTestListContinuationWithFilter(t, newTestClient(t))
}

func TestListInconsistentContinuation(t *testing.T) {
	generic.RunTestListInconsistentContinuation(t, newTestClient(t))
}

func TestPrefix(t *testing.T) {
	generic.RunTestPrefix(t, newTestClient(t))
}

func TestConsistentList(t *testing.T) {
	generic.RunTestConsistentList(t, newTestClient(t))
}

func TestCount(t *testing.T) {
	generic.RunTestCount(t, newTestClient(t))
}

func TestLeaseMaxObjectCount(t *testing.T) {
	etcdClient := testserver.RunEtcd(t, nil)
	recorder := &clientRecorder{KV: etcdClient.KV}
	etcdClient.KV = recorder
	client := &internalTestClient{
		client: &client{
			Client: etcdClient,
			leaseManager: newDefaultLeaseManager(etcdClient, LeaseManagerConfig{
				ReuseDurationSeconds: defaultLeaseReuseDurationSeconds,
				MaxObjectCount:       2,
			}),
		},
		ReadRecorder: recorder,
	}
	ctx, store := generic.TestSetup(client)

	obj := &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}
	out := &example.Pod{}

	testCases := []struct {
		key                 string
		expectAttachedCount int64
	}{
		{
			key:                 "testkey1",
			expectAttachedCount: 1,
		},
		{
			key:                 "testkey2",
			expectAttachedCount: 2,
		},
		{
			key: "testkey3",
			// We assume each time has 1 object attached to the lease
			// so after granting a new lease, the recorded count is set to 1
			expectAttachedCount: 1,
		},
	}

	for _, tc := range testCases {
		err := store.Create(ctx, tc.key, obj, out, 120)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		if client.leaseManager.leaseAttachedObjectCount != tc.expectAttachedCount {
			t.Errorf("Lease manager recorded count %v should be %v", client.leaseManager.leaseAttachedObjectCount, tc.expectAttachedCount)
		}
	}
}
