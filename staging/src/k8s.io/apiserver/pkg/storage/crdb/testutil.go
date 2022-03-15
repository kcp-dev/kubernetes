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

package crdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cockroachdb/apd"
	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/value"
)

func TestBootstrapper() storage.TestBootstrapper {
	return &crdbTestBootstrapper{}
}

type crdbTestBootstrapper struct{}

func (e *crdbTestBootstrapper) InterfaceForClient(t *testing.T, client storage.InternalTestClient, codec runtime.Codec, newFunc func() runtime.Object, prefix string, groupResource schema.GroupResource, transformer value.Transformer, pagingEnabled bool) storage.InternalTestInterface {
	pool, ok := client.(*crdbTestClient)
	if !ok {
		t.Fatalf("got a %T, not crdb test client", client)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
	})
	store := newStore(ctx, pool, codec, newFunc, prefix, groupResource, transformer, pagingEnabled)
	return &crdbTestInterface{store: store}
}

func (e *crdbTestBootstrapper) Setup(t *testing.T, codec runtime.Codec, newFunc func() runtime.Object, prefix string, groupResource schema.GroupResource, transformer value.Transformer, pagingEnabled bool) (context.Context, storage.InternalTestInterface, storage.InternalTestClient) {
	ts, err := testserver.NewTestServer()
	if err != nil {
		t.Fatalf("failed to start crdb: %v", err)
	}
	t.Cleanup(func() {
		ts.Stop()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
	})
	cfg, err := pgxpool.ParseConfig(ts.PGURL().String())
	if err != nil {
		t.Fatalf("failed to parse test connection: %v", err)
	}
	cfg.ConnConfig.LogLevel = pgx.LogLevelTrace
	cfg.ConnConfig.Logger = NewLogger(t)
	client, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect to crdb: %v", err)
	}
	recordingClient := &recordingRowQuerier{Pool: client}

	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS k8s
			(
				key VARCHAR(512) NOT NULL,
				value BLOB NOT NULL,
				cluster VARCHAR(256) NOT NULL,
				namespace VARCHAR(63),
				name VARCHAR(127),
				api_group VARCHAR(63),
				api_version VARCHAR(63),
				api_resource VARCHAR(63),
				PRIMARY KEY (key, cluster)
			);`,
		// enable watches
		`SET CLUSTER SETTING kv.rangefeed.enabled = true;`,
		// set the latency floor for events
		`SET CLUSTER SETTING changefeed.experimental_poll_interval = '0.2s';`,
		// TODO: indices for access patterns
		//`CREATE INDEX k8s_gvr_index ON k8s (group, version, resource)`,
	} {
		if _, err := client.Exec(ctx, stmt); err != nil {
			t.Fatalf("error initializing the database: %v", err)
		}
	}
	store := newStore(ctx, recordingClient, codec, newFunc, prefix, groupResource, transformer, pagingEnabled)

	clusterName := "admin"
	ctx = request.WithCluster(ctx, request.Cluster{Name: clusterName})
	ctx = request.WithRequestInfo(ctx, &request.RequestInfo{
		APIGroup:   "",
		APIVersion: "v1",
		Namespace:  "ns",
		Resource:   "pods",
		Name:       "foo",
	})
	return ctx, &crdbTestInterface{store: store}, &crdbTestClient{recordingRowQuerier: recordingClient}
}

func NewRawClient(pool *pgxpool.Pool) storage.InternalTestClient {
	recordingClient := &recordingRowQuerier{Pool: pool}
	return &crdbTestClient{recordingRowQuerier: recordingClient}
}

type crdbTestClient struct {
	*recordingRowQuerier
}

func (e *crdbTestClient) RawWatch(ctx context.Context, key string, revision int64) storage.WatchChan {
	watchChan := make(chan storage.WatchResponse, 20)
	go func() {
		providedTimestamp, err := toHybridLogicalClock(revision)
		if err != nil {
			watchChan <- storage.WatchResponse{Error: fmt.Errorf("failed to parse initial revision: %w", err)}
			return
		}
		var initialWatchTimestamp apd.Decimal
		condition, err := apd.BaseContext.Add(&initialWatchTimestamp, providedTimestamp, apd.New(1, 0))
		if err != nil {
			watchChan <- storage.WatchResponse{Error: fmt.Errorf("failed to determine initial watch revision: %v", err)}
			return
		}
		if _, err := condition.GoError(apd.DefaultTraps); err != nil {
			watchChan <- storage.WatchResponse{Error: fmt.Errorf("failed to determine initial watch revision: %v", err)}
			return
		}
		options := []string{
			"updated",
			"mvcc_timestamp",
			fmt.Sprintf("cursor='%s'", providedTimestamp),
		}
		query := fmt.Sprintf(`EXPERIMENTAL CHANGEFEED FOR k8s WITH %s;`, strings.Join(options, ","))
		rows, err := e.Query(ctx, query)
		if err != nil {
			watchChan <- storage.WatchResponse{Error: fmt.Errorf("failed to create changefeed: %w", err)}
		}
		defer func() {
			rows.Close()
		}()
		for rows.Next() {
			if err := rows.Err(); err != nil {
				watchChan <- storage.WatchResponse{Error: fmt.Errorf("failed to read changefeed row: %w", err)}
				return
			}

			values := rows.RawValues()
			if len(values) != 3 {
				watchChan <- storage.WatchResponse{Error: fmt.Errorf("expected 3 values in changefeed row, got %d", len(values))}
				return
			}
			// values upacks into (tableName, primaryKey, rowData)
			// tableName = name
			// primaryKey = ["array", "of", "values"]
			// rowData = {...}
			data := values[2]

			var evt changefeedEvent
			if err := json.Unmarshal(data, &evt); err != nil {
				watchChan <- storage.WatchResponse{Error: fmt.Errorf("failed to deserialize changefeed row: %w", err)}
				return
			}

			if evt.MVCCTimestamp != nil {
				resourceVersion, err := toResourceVersion(evt.MVCCTimestamp)
				if err != nil {
					watchChan <- storage.WatchResponse{Error: err}
					return
				}
				watchChan <- storage.WatchResponse{Revision: resourceVersion}
				continue
			}
		}
	}()
	return watchChan
}

type recordingRowQuerier struct {
	reads uint64
	*pgxpool.Pool
}

func (r *recordingRowQuerier) BeginTxFunc(ctx context.Context, txOptions pgx.TxOptions, f func(pgx.Tx) error) error {
	atomic.AddUint64(&r.reads, 1)
	return r.Pool.BeginTxFunc(ctx, txOptions, f)
}

func (e *crdbTestClient) Reads() uint64 {
	return atomic.LoadUint64(&e.reads)
}

func (e *crdbTestClient) ResetReads() {
	atomic.StoreUint64(&e.reads, 0)
}

func (e *crdbTestClient) RawPut(ctx context.Context, key, data string) (revision int64, err error) {
	return create(ctx, e.Pool, key, []byte(data))
}

func (e *crdbTestClient) RawGet(ctx context.Context, key string) ([]byte, bool, error) {
	clusterName := "admin"
	var v []byte
	err := e.QueryRow(ctx, `SELECT value FROM k8s WHERE key=$1 AND cluster=$2;`, key, clusterName).Scan(&v)
	notFound := errors.Is(err, pgx.ErrNoRows)
	if notFound {
		err = nil
	}
	return v, notFound, err
}

func (e *crdbTestClient) RawCompact(ctx context.Context, revision int64) error {
	// TODO: is there something better than this? no obvious way to trigger a compaction, and definitely not for a specific revision.
	// TODO: maybe restore from backup?
	hybridLogcialTimestamp, err := toHybridLogicalClock(revision)
	if err != nil {
		return fmt.Errorf("failed to determine hybrid logical clock when handling compaction event: %w", err)
	}
	revisionTimestampNanoseconds, err := hybridLogcialTimestamp.Int64()
	if err != nil && !strings.Contains(err.Error(), "has fractional part") {
		// having a fractional part means we have a logical clock, but that only disambiguates
		// between ns, so it's ok to ignore that error
		return err
	}
	revisionTimestamp := time.Unix(0, revisionTimestampNanoseconds)
	// incoming revision will be a ns timestamp, so why don't we fetch the current time and
	// figure out how long the GC interval would have to be so we would no longer hold the
	// incoming revision (as of now)
	var clusterHybridLogicalTimestamp apd.Decimal
	if err := e.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&clusterHybridLogicalTimestamp); err != nil {
		return err
	}
	clusterTimestampNanoseconds, err := clusterHybridLogicalTimestamp.Int64()
	if err != nil && !strings.Contains(err.Error(), "has fractional part") {
		// having a fractional part means we have a logical clock, but that only disambiguates
		// between ns, so it's ok to ignore that error
		return err
	}
	clusterTimestamp := time.Unix(0, clusterTimestampNanoseconds)

	age := clusterTimestamp.Sub(revisionTimestamp)
	fmt.Printf("COMPACTION: %s - %s = %s\n", clusterTimestamp, revisionTimestamp, age)
	if age < time.Second {
		age = time.Second
	}
	if _, err := e.Exec(ctx, `ALTER TABLE k8s CONFIGURE ZONE USING gc.ttlseconds = $1;`, age.Seconds()); err != nil {
		return err
	}

	// in order to wait for the compaction, ask for the revision and wait until it's out of scope
	return wait.Poll(500*time.Millisecond, 10*age, func() (done bool, err error) {
		var key string
		if err := e.QueryRow(ctx, fmt.Sprintf(`SELECT key FROM k8s AS OF SYSTEM TIME %s LIMIT 1;`, hybridLogcialTimestamp)).Scan(&key); err != nil {
			if isGarbageCollectionError(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}

type crdbTestInterface struct {
	*store
}

func (e *crdbTestInterface) PathPrefix() string {
	return e.pathPrefix
}

func (e *crdbTestInterface) UpdateTransformer(f func(transformer value.Transformer) value.Transformer) {
	e.transformer = f(e.transformer)
}

func (e *crdbTestInterface) Decode(raw []byte) (runtime.Object, error) {
	return runtime.Decode(e.codec, raw)
}

// apparently the logging adapter is wrong ... ?

// TestingLogger interface defines the subset of testing.TB methods used by this
// adapter.
type TestingLogger interface {
	Log(args ...interface{})
	Helper()
}

type Logger struct {
	l TestingLogger
}

func NewLogger(l TestingLogger) *Logger {
	return &Logger{l: l}
}

func (l *Logger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	l.l.Helper()
	logArgs := make([]interface{}, 0, 2+len(data))
	logArgs = append(logArgs, level, msg)
	for k, v := range data {
		logArgs = append(logArgs, fmt.Sprintf("%s=%v", k, v))
	}
	l.l.Log(logArgs...)
}
