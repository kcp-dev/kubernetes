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

package crdb

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cockroachdb/apd"
	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"
)

func newFromStringOrDie(input string) *apd.Decimal {
	d, c, e := apd.NewFromString(input)
	if c.Any() {
		panic(c.String())
	}
	if e != nil {
		panic(e)
	}
	return d
}

func TestToResourceVersion(t *testing.T) {
	var testCases = []struct {
		name      string
		input     *apd.Decimal
		output    int64
		expectErr bool
	}{
		{
			name:   "no logical clock",
			input:  newFromStringOrDie("1640995200000000001"), // datum + 1
			output: 16,
		},
		{
			name:   "large physical clock",
			input:  newFromStringOrDie(strconv.Itoa(1<<20 + 1640995200000000000)),
			output: 1 << 24,
		},
		{
			name:   "with logical clock",
			input:  newFromStringOrDie("1640995200000000001.0000000001"),
			output: 17,
		},
		{
			name:      "with physical clock overflow",
			input:     newFromStringOrDie("2217455952303423488.0000000000"),
			expectErr: true,
		},
		{
			name:      "with logical clock overflow",
			input:     newFromStringOrDie("1640995200000000001.0000000016"),
			expectErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := toResourceVersion(testCase.input)
			if err == nil && testCase.expectErr {
				t.Errorf("%s: expected an error but got none", testCase.name)
			}
			if err != nil && !testCase.expectErr {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, err)
			}
			if actual != testCase.output {
				t.Errorf("%s: expected %d but got %d", testCase.name, testCase.output, actual)
			}
		})
	}
}

func TestToHybridLogicalClock(t *testing.T) {
	var testCases = []struct {
		name      string
		input     int64
		output    *apd.Decimal
		expectErr bool
	}{
		{
			name:   "no logical clock",
			input:  16,
			output: newFromStringOrDie("1640995200000000001"),
		},
		{
			name:   "large physical clock",
			input:  1 << 24,
			output: newFromStringOrDie(strconv.Itoa(1<<20 + 1640995200000000000)),
		},
		{
			name:   "with logical clock",
			input:  17,
			output: newFromStringOrDie("1640995200000000001.0000000001"),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := toHybridLogicalClock(testCase.input)
			if err == nil && testCase.expectErr {
				t.Errorf("%s: expected an error but got none", testCase.name)
			}
			if err != nil && !testCase.expectErr {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, err)
			}
			if actual.Cmp(testCase.output) != 0 {
				t.Errorf("%s: expected %s but got %s", testCase.name, testCase.output.String(), actual.String())
			}
		})
	}
}

func TestResourceVersionRoundTripping(t *testing.T) {
	var testCases = []struct {
		name  string
		input *apd.Decimal
	}{
		{
			name:  "no logical clock",
			input: newFromStringOrDie("1640995200000000001"),
		},
		{
			name:  "large physical clock",
			input: newFromStringOrDie(strconv.Itoa(1<<20 + 1640995200000000000)),
		},
		{
			name:  "with logical clock",
			input: newFromStringOrDie("1640995200000000001.0000000001"),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rv, err := toResourceVersion(testCase.input)
			if err != nil {
				t.Fatalf("error getting rv: %v", err)
			}
			hlc, err := toHybridLogicalClock(rv)
			if err != nil {
				t.Fatalf("error getting hlc: %v", err)
			}
			if hlc.Cmp(testCase.input) != 0 {
				t.Fatalf("invalid round-tripped hlc: before: %s, after: %s", testCase.input.String(), hlc.String())
			}
		})
	}
}

func TestWhereClause(t *testing.T) {
	var testCases = []struct {
		name    string
		offset  int
		cluster *request.Cluster
		info    *request.RequestInfo
		clause  string
		args    []interface{}
	}{
		{
			name:    "with everything",
			offset:  3,
			cluster: &request.Cluster{Name: "logical"},
			info: &request.RequestInfo{
				APIGroup:   "items.thing.com",
				APIVersion: "v2",
				Namespace:  "ns",
				Resource:   "things",
				Name:       "this",
			},
			clause: " AND cluster=$3 AND name=$4 AND namespace=$5 AND api_version=$6 AND api_group=$7 AND api_resource=$8",
			args:   []interface{}{"logical", "this", "ns", "v2", "items.thing.com", "things"},
		},
		{
			name:    "wildcard cluster",
			cluster: &request.Cluster{Name: "logical", Wildcard: true},
			info: &request.RequestInfo{
				APIGroup:   "items.thing.com",
				APIVersion: "v2",
				Namespace:  "ns",
				Resource:   "things",
				Name:       "this",
			},
			clause: " AND name=$1 AND namespace=$2 AND api_version=$3 AND api_group=$4 AND api_resource=$5",
			args:   []interface{}{"this", "ns", "v2", "items.thing.com", "things"},
		},
		{
			name:    "wildcard cluster and cross-namespace",
			cluster: &request.Cluster{Name: "logical", Wildcard: true},
			info: &request.RequestInfo{
				APIGroup:   "items.thing.com",
				APIVersion: "v2",
				Namespace:  v1.NamespaceAll,
				Resource:   "things",
				Name:       "this",
			},
			clause: " AND name=$1 AND api_version=$2 AND api_group=$3 AND api_resource=$4",
			args:   []interface{}{"this", "v2", "items.thing.com", "things"},
		},
		{
			name:    "no info",
			cluster: &request.Cluster{Name: "logical"},
			clause:  " AND cluster=$1",
			args:    []interface{}{"logical"},
		},
		{
			name: "no cluster",
			info: &request.RequestInfo{
				APIGroup:   "items.thing.com",
				APIVersion: "v2",
				Namespace:  "ns",
				Resource:   "things",
				Name:       "this",
			},
			clause: " AND name=$1 AND namespace=$2 AND api_version=$3 AND api_group=$4 AND api_resource=$5",
			args:   []interface{}{"this", "ns", "v2", "items.thing.com", "things"},
		},
		{
			name:   "no anything",
			clause: "",
			args:   nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			clause, args := whereClause(testCase.offset, testCase.cluster, testCase.info)
			if diff := cmp.Diff(clause, testCase.clause); diff != "" {
				t.Errorf("incorrect clause: %s", diff)
			}
			if diff := cmp.Diff(args, testCase.args); diff != "" {
				t.Errorf("incorrect args: %s", diff)
			}
		})
	}
}

func TestWatchConsistency(t *testing.T) {
	// CRDB changefeeds are different from etcd watches in that monotonicity is only guaranteed for
	// the hybrid-logical clock of events *for a given row* - events for separate rows may come in out-
	// of-order. Previous work in KCP has shown that relying on comparison of resourceVersion between
	// objects in k8s is fraught with error as it is, as actors like storage migration or encryption
	// key rotation may alter the resourceVersion of objects in random manners, removing any ability
	// for clients to infer causality.
	// In any case, the e2e is not strictly applicable to CRDB. However, if we sort the events that
	// come out of changefeeds by their HLC, we can show that the same principles of consistent watching
	// apply to CRDB changefeeds.
	ts, err := testserver.NewTestServer()
	if err != nil {
		t.Fatalf("failed to start crdb: %v", err)
	}
	defer func() {
		ts.Stop()
	}()

	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(ts.PGURL().String())
	if err != nil {
		t.Fatalf("failed to parse test connection: %v", err)
	}
	cfg.ConnConfig.LogLevel = pgx.LogLevelTrace
	cfg.ConnConfig.Logger = NewLogger(t)
	cfg.MaxConns = 128
	client, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect to crdb: %v", err)
	}

	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS test
			(
				key INTEGER PRIMARY KEY,
				value INTEGER NOT NULL
			);`,
		// enable changefeeds
		`SET CLUSTER SETTING kv.rangefeed.enabled = true;`,
		// set the latency floor for events
		`SET CLUSTER SETTING changefeed.experimental_poll_interval = '0.2s';`,
		// remove throttling
		`SET CLUSTER SETTING changefeed.node_throttle_config = '{"MessageRage":0,"MessageBurst":0,"ByteRate":0,"ByteBurst":0,"FlushRate":0,"FlushBurst":0}';`,
	} {
		if _, err := client.Exec(ctx, stmt); err != nil {
			t.Fatalf("error initializing the database: %v", err)
		}
	}

	var initialClusterTimestamp apd.Decimal
	if err := client.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&initialClusterTimestamp); err != nil {
		t.Fatalf("failed to read initial cluster logical timestamp: %v", err)
	}
	t.Logf("Initial cluster timestamp: %s", initialClusterTimestamp.String())

	const (
		createEvent = iota
		updateEvent
		deleteEvent
	)

	numUpdates := 25
	var existing []int
	for i := 0; i < numUpdates; i++ {
		op := rand.Intn(3)
		if len(existing) == 0 {
			op = createEvent
		}

		switch op {
		case createEvent:
			key := i
			if _, err := client.Exec(ctx, `INSERT INTO test (key, value) VALUES ($1, $2);`, key, 0); err != nil {
				t.Fatalf("unexpected error while inserting new row: %v", err)
			}
			existing = append(existing, key)
		case updateEvent:
			key := existing[rand.Intn(len(existing))]
			if _, err := client.Exec(ctx, `UPDATE test SET value = value + 1 WHERE key=$1;`, key); err != nil {
				t.Fatalf("unexpected error while updating row: %v", err)
			}
		case deleteEvent:
			idx := rand.Intn(len(existing))
			key := existing[idx]
			if _, err := client.Exec(ctx, `DELETE FROM test WHERE key=$1;`, key); err != nil {
				t.Fatalf("unexpected error while removing row: %v", err)
			}
			existing = append(existing[:idx], existing[idx+1:]...)
		default:
			t.Fatalf("invalid operation %d", op)
		}
	}

	var finalClusterTimestamp apd.Decimal
	if err := client.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&finalClusterTimestamp); err != nil {
		t.Fatalf("failed to read initial cluster logical timestamp: %v", err)
	}
	t.Logf("Final cluster timestamp: %s", finalClusterTimestamp.String())

	lock := sync.Mutex{}
	var idx int
	sink := make([][]kvEvent, numUpdates+1)
	order := map[string]int{}
	wg := sync.WaitGroup{}

	into := make([]chan kvEvent, numUpdates+1)
	for i := 0; i < numUpdates+1; i++ {
		into[i] = make(chan kvEvent)
	}
	launch(t, ctx, client, &initialClusterTimestamp, &finalClusterTimestamp, &idx, idx, &wg, into, &sink, order, &lock)

	done := make(chan interface{})
	go func() {
		wg.Wait()
		done <- nil
	}()

	select {
	case <-done:
		for i := range into {
			close(into[i])
		}
	case <-time.After(10 * time.Second):
		logrus.Error("timed out waiting for changefeeds")
	}

	lock.Lock()
	defer lock.Unlock()
	// CRDB does not guarantee ordering between rows, just within them, so, first we sort the events we got in the reference
	sort.Slice(sink[0], func(x, y int) bool {
		return sink[0][x].timestamp.Cmp(sink[0][y].timestamp) < 0
	})
	// then, we can reorder each child changefeed to figure out which subset it should have seen
	reorderedSink := make([][]kvEvent, numUpdates+1)
	reorderedSink[0] = sink[0]
	for i, item := range sink[0] {
		idx := order[item.timestamp.String()]
		reorderedSink[i+1] = sink[idx]
	}

	// then, we can sort *all* events for sensible outcomes
	for i := range reorderedSink {
		sort.Slice(reorderedSink[i], func(x, y int) bool {
			return reorderedSink[i][x].timestamp.Cmp(reorderedSink[i][y].timestamp) < 0
		})
	}
	cursor := func(i int) string {
		if i == 0 {
			return initialClusterTimestamp.String()
		}
		return reorderedSink[0][i-1].timestamp.String()
	}

	for i := range reorderedSink {
		for j := range reorderedSink[i] {
			if reorderedSink[i][j].timestamp.Cmp(reorderedSink[0][i].timestamp) < 0 {
				logrus.Errorf("changefeed %d (cursor=%s) saw an event at %s, before the cursor", i, cursor(i), reorderedSink[i][j].timestamp.String())
			}
		}
	}

	formattedSink := make([][]string, len(reorderedSink))
	for i := range reorderedSink {
		formattedSink[i] = make([]string, len(reorderedSink[i]))
		for j := range reorderedSink[i] {
			formattedSink[i][j] = reorderedSink[i][j].String()
		}
	}

	reference := formattedSink[0]
	for i, updates := range formattedSink {
		id := fmt.Sprintf("changefeed %d (cursor=%s) ", i, cursor(i))
		if actual, expected := len(updates), numUpdates-i; actual != expected {
			logrus.Errorf("%sgot %d events, expected %d", id, actual, expected)
		}
		if len(updates) == 0 {
			continue
		}
		if i == 0 {
			continue
		}
		startingIndex := -1
		for j, item := range reference {
			if item == updates[0] {
				startingIndex = j
			}
		}
		if startingIndex == -1 {
			logrus.Errorf("%sstarted seeing events at timestamp %q, but the reference watcher never saw that version!", id, updates[0])
			continue
		}
		if startingIndex != i {
			logrus.Errorf("%sstarted seeing events at index %d, expected %d", id, startingIndex, i)
		}
		if diff := cmp.Diff(reference[i:], updates); diff != "" {
			logrus.Errorf("%sgot incorrect ordering for events: %v", id, diff)
		}
	}
}

type kvEvent struct {
	timestamp *apd.Decimal
	action    string
}

func (e kvEvent) String() string {
	return fmt.Sprintf("%s@%s", e.action, e.timestamp)
}

func launch(t *testing.T, ctx context.Context, client *pgxpool.Pool, start, end *apd.Decimal, idx *int, i int, wg *sync.WaitGroup, into []chan kvEvent, sink *[][]kvEvent, order map[string]int, lock *sync.Mutex) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		changefeed(t, ctx, start, end, client, into[i])
	}()
	go func() {
		for evt := range into[i] {
			lock.Lock()
			(*sink)[i] = append((*sink)[i], evt)
			if i == 0 {
				*idx++
				order[evt.timestamp.String()] = *idx
				launch(t, ctx, client, evt.timestamp, end, idx, *idx, wg, into, sink, order, lock)
			}
			lock.Unlock()
		}
	}()
}

func changefeed(t *testing.T, ctx context.Context, start, end *apd.Decimal, client *pgxpool.Pool, into chan<- kvEvent) {
	options := []string{
		"updated",
		"diff",
		"mvcc_timestamp",
		fmt.Sprintf("cursor='%s'", start.String()),
		"resolved='1s'",
	}
	query := fmt.Sprintf(`EXPERIMENTAL CHANGEFEED FOR test WITH %s;`, strings.Join(options, ","))
	rows, err := client.Query(ctx, query)
	if err != nil {
		t.Fatalf("failed to create changefeed: %v", err)
	}
	defer func() {
		go func() {
			rows.Close()
		}()
	}()
	for rows.Next() {
		if err := rows.Err(); err != nil {
			t.Fatalf("failed to read changefeed row: %v", err)
		}

		values := rows.RawValues()
		if len(values) != 3 {
			t.Fatalf("expected 3 values in changefeed row, got %d", len(values))
		}

		// values upacks into (tableName, primaryKey, rowData)
		data := values[2]

		type row struct {
			Key   int64 `json:"key,omitempty"`
			Value int64 `json:"value,omitempty"`
		}

		type changefeedEvent struct {
			Updated  *apd.Decimal `json:"updated,omitempty"`
			Resolved *apd.Decimal `json:"resolved,omitempty"`
			Before   *row         `json:"before,omitempty"`
			After    *row         `json:"after,omitempty"`
		}
		var evt changefeedEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			t.Fatalf("failed to deserialize changefeed row: %v", err)
		}

		if evt.Resolved != nil {
			if evt.Resolved.Cmp(end) == 1 {
				// we've seen everything we need to see
				return
			}
		} else if evt.Updated != nil {
			var action string
			switch {
			case evt.Before == nil && evt.After != nil:
				action = fmt.Sprintf("INSERT(%d=%d)", evt.After.Key, evt.After.Value)
			case evt.Before != nil && evt.After != nil:
				action = fmt.Sprintf("UPDATE(%d=%d->%d)", evt.After.Key, evt.Before.Value, evt.After.Value)
			case evt.Before != nil && evt.After == nil:
				action = fmt.Sprintf("DELETE(%d)", evt.Before.Key)
			}
			into <- kvEvent{
				timestamp: evt.Updated,
				action:    action,
			}
		}
	}
}
