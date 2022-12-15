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
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"k8s.io/apiserver/pkg/storage/generic"
)

const (
	rowTTLUnsupported = "row-level TTLs are not supported yet in CRDB"
)

type internalTestClient struct {
	*testClient
	generic.ReadRecorder
}

var _ generic.InternalTestClient = &internalTestClient{}

func newTestClient(t *testing.T) *internalTestClient {
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
	pool, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect to crdb: %v", err)
	}

	if err := InitializeDB(ctx, pool, 0); err != nil {
		t.Fatal(err)
	}

	recordingClient := &recordingRowQuerier{pool: pool}

	return &internalTestClient{
		testClient: &testClient{
			client: &client{
				pool:            recordingClient,
				changefeedCache: initialize(ctx, pool),
			},
		},
		ReadRecorder: recordingClient,
	}
}

type recordingRowQuerier struct {
	reads uint64
	pool
}

func (r *recordingRowQuerier) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	atomic.AddUint64(&r.reads, 1)
	return r.pool.BeginTx(ctx, txOptions)
}

func (r *recordingRowQuerier) Reads() uint64 {
	return atomic.LoadUint64(&r.reads)
}

func (r *recordingRowQuerier) ResetReads() {
	atomic.StoreUint64(&r.reads, 0)
}

func (c *internalTestClient) Watch(ctx context.Context, key string, recursive, progressNotify bool, revision int64) <-chan *generic.WatchResponse {
	return watchFromChangefeed(ctx, c.pool, key, recursive, progressNotify, revision)
}

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

func TestCreate(t *testing.T) {
	generic.RunTestCreate(t, newTestClient(t))
}

func TestCreateWithTTL(t *testing.T) {
	t.Skip(rowTTLUnsupported)
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
	t.Skip(rowTTLUnsupported)
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
