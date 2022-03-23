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

	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apiserver/pkg/storage/generic"
)

func NewTestClient(c *clientv3.Client, config LeaseManagerConfig) generic.TestClient{
	return &testClient{
		client: NewClient(c, config),
	}
}

type testClient struct {
	*client
}

func (c *testClient) Compact(ctx context.Context, revision int64) error {
	_, err := c.client.KV.Compact(ctx, revision, clientv3.WithCompactPhysical())
	return err
}

var _ generic.TestClient = &testClient{}