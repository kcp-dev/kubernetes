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

package factory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	"k8s.io/apiserver/pkg/storage/crdb"

	"k8s.io/apimachinery/pkg/runtime"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/egressselector"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/apiserver/pkg/storage/value"
)

func newCRDBHealthCheck(ctx context.Context, c storagebackend.Config) (func() error, error) {
	// constructing the etcd v3 client blocks and times out if etcd is not available.
	// retry in a loop in the background until we successfully create the client, storing the client or error encountered

	clientValue := &atomic.Value{}

	clientErrMsg := &atomic.Value{}
	clientErrMsg.Store("etcd client connection not yet established")

	go wait.PollUntil(time.Second, func() (bool, error) {
		client, err := newCRDBClient(ctx, c.Transport)
		if err != nil {
			clientErrMsg.Store(err.Error())
			return false, nil
		}
		clientValue.Store(client)
		clientErrMsg.Store("")
		return true, nil
	}, wait.NeverStop)

	return func() error {
		if errMsg := clientErrMsg.Load().(string); len(errMsg) > 0 {
			return fmt.Errorf(errMsg)
		}
		client := clientValue.Load().(*pgxpool.Pool)
		healthcheckTimeout := storagebackend.DefaultHealthcheckTimeout
		if c.HealthcheckTimeout != time.Duration(0) {
			healthcheckTimeout = c.HealthcheckTimeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeout)
		defer cancel()
		// TODO: use health cluster api https://www.cockroachlabs.com/docs/api/cluster/v2#operation/health
		var err error
		fmt.Println(ctx, client)
		return fmt.Errorf("error getting data from etcd: %v", err)
	}, nil
}

func newCRDBClient(ctx context.Context, c storagebackend.TransportConfig) (*pgxpool.Pool, error) {
	tlsInfo := transport.TLSInfo{
		CertFile:      c.CertFile,
		KeyFile:       c.KeyFile,
		TrustedCAFile: c.TrustedCAFile,
	}
	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, err
	}
	// NOTE: Client relies on nil tlsConfig
	// for non-secure connections, update the implicit variable
	if len(c.CertFile) == 0 && len(c.KeyFile) == 0 && len(c.TrustedCAFile) == 0 {
		tlsConfig = nil
	}
	networkContext := egressselector.Etcd.AsNetworkContext()
	var egressDialer utilnet.DialFunc
	if c.EgressLookup != nil {
		egressDialer, err = c.EgressLookup(networkContext)
		if err != nil {
			return nil, err
		}
	}

	cfg, err := pgxpool.ParseConfig(c.ServerList[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse test connection: %v", err)
	}
	cfg.ConnConfig.TLSConfig = tlsConfig
	if egressDialer != nil {
		cfg.ConnConfig.DialFunc = pgconn.DialFunc(egressDialer)
	}
	return pgxpool.ConnectConfig(ctx, cfg)
}

func newCRDBStorage(ctx context.Context, c storagebackend.ConfigForResource, newFunc func() runtime.Object) (storage.Interface, DestroyFunc, error) {
	stopCompactor, err := startCompactorOnce(c.Transport, c.CompactionInterval)
	if err != nil {
		return nil, nil, err
	}

	client, err := newCRDBClient(ctx, c.Transport)
	if err != nil {
		stopCompactor()
		return nil, nil, err
	}

	// no way to monitor size: https://github.com/cockroachdb/cockroach/issues/20712

	var once sync.Once
	destroyFunc := func() {
		// we know that storage destroy funcs are called multiple times (due to reuse in subresources).
		// Hence, we only destroy once.
		// TODO: fix duplicated storage destroy calls higher level
		once.Do(func() {
			stopCompactor()
			client.Close()
		})
	}
	transformer := c.Transformer
	if transformer == nil {
		transformer = value.IdentityTransformer
	}
	return crdb.New(client, c.Codec, newFunc, c.Prefix, c.GroupResource, transformer, c.Paging), destroyFunc, nil
}
