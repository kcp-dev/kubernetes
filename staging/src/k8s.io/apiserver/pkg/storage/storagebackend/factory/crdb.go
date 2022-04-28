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

package factory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	"k8s.io/apiserver/pkg/storage/crdb"
	"k8s.io/apiserver/pkg/storage/generic"
	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/runtime"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/egressselector"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/apiserver/pkg/storage/value"
)

func newCRDBHealthCheck(ctx context.Context, c storagebackend.Config) (func() error, error) {
	clientValue := &atomic.Value{}

	clientErrMsg := &atomic.Value{}
	clientErrMsg.Store("crdb client connection not yet established")

	go wait.PollImmediateUntil(time.Second, func() (bool, error) {
		client, err := newCRDBClient(ctx, c.Transport)
		if err != nil {
			klog.Errorf("failed to get crdb client: %v", err)
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
		err := client.Ping(ctx)
		if err == nil {
			return nil
		}
		return fmt.Errorf("error getting data from crdb: %v", err)
	}, nil
}

func NewCRDBTestClient(ctx context.Context, c storagebackend.TransportConfig) (generic.TestClient, error) {
	crdbClient, err := newCRDBClient(ctx, c)
	if err != nil {
		return nil, err
	}
	return crdb.NewTestClient(ctx, crdbClient), nil
}

func newCRDBClient(ctx context.Context, c storagebackend.TransportConfig) (*pgxpool.Pool, error) {
	tlsInfo := transport.TLSInfo{ // TODO: stop using the etcd lib here
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
	networkContext := egressselector.CRDB.AsNetworkContext()
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
	if tlsConfig != nil {
		tlsConfig.ServerName = cfg.ConnConfig.Config.Host
	}
	cfg.ConnConfig.TLSConfig = tlsConfig
	if egressDialer != nil {
		cfg.ConnConfig.DialFunc = pgconn.DialFunc(egressDialer)
	}
	cfg.ConnConfig.LogLevel = pgx.LogLevelWarn
	cfg.ConnConfig.Logger = NewLogger()
	cfg.MaxConns = 8192 // TODO: we need a routine to poll `Stat()` and expose metrics, so users would know when we're close to hitting the max
	return pgxpool.ConnectConfig(ctx, cfg)
}

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	var fields []interface{}
	for k, v := range data {
		fields = append(fields, k)
		fields = append(fields, v)
	}
	fields = append(fields, "PGX_LOG_LEVEL")
	fields = append(fields, level)
	switch level {
	case pgx.LogLevelTrace:
		klog.V(4).InfoS(msg, fields...)
	case pgx.LogLevelDebug:
		klog.V(3).InfoS(msg, fields...)
	case pgx.LogLevelInfo:
		klog.V(2).InfoS(msg, fields...)
	case pgx.LogLevelWarn:
		klog.ErrorS(nil, msg, fields...)
	case pgx.LogLevelError:
		klog.ErrorS(nil, msg, fields...)
	default:
		klog.ErrorS(nil, msg, fields...)
	}
}

var once = sync.Once{}

func newCRDBStorage(c storagebackend.ConfigForResource, enableCaching bool, newFunc func() runtime.Object) (storage.Interface, DestroyFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	client, err := newCRDBClient(ctx, c.Transport)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	var initErr error
	once.Do(func() {
		initErr = crdb.InitializeDB(ctx, client, c.CompactionInterval)
	})
	if initErr != nil {
		cancel()
		return nil, nil, initErr
	}

	// TODO: no way to monitor size: https://github.com/cockroachdb/cockroach/issues/20712

	var once sync.Once
	destroyFunc := func() {
		// we know that storage destroy funcs are called multiple times (due to reuse in subresources).
		// Hence, we only destroy once.
		// TODO: fix duplicated storage destroy calls higher level
		once.Do(func() {
			cancel()
			client.Close()
		})
	}
	transformer := c.Transformer
	if transformer == nil {
		transformer = value.IdentityTransformer
	}
	return crdb.New(crdb.NewClient(ctx, enableCaching, client), c.Codec, newFunc, c.Prefix, c.GroupResource, transformer, c.Paging), destroyFunc, nil
}
