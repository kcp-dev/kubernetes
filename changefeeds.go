package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/apd"
	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

func main() {
	ts, err := testserver.NewTestServer()
	if err != nil {
		logrus.WithError(err).Fatal("failed to start crdb")
	}
	defer func() {
		ts.Stop()
	}()

	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(ts.PGURL().String())
	if err != nil {
		logrus.WithError(err).Fatal("failed to parse test connection")
	}
	cfg.ConnConfig.LogLevel = pgx.LogLevelTrace
	cfg.ConnConfig.Logger = logrusadapter.NewLogger(logrus.New())
	cfg.MaxConns = 128
	client, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		logrus.WithError(err).Fatal("failed to connect to crdb")
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
			logrus.WithError(err).Fatal("error initializing the database")
		}
	}

	var initialClusterTimestamp apd.Decimal
	if err := client.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&initialClusterTimestamp); err != nil {
		logrus.WithError(err).Fatal("failed to read initial cluster logical timestamp")
	}
	logrus.Infof("Initial cluster timestamp: %s", initialClusterTimestamp.String())

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
				logrus.WithError(err).Fatal("unexpected error while inserting new row")
			}
			existing = append(existing, key)
		case updateEvent:
			key := existing[rand.Intn(len(existing))]
			if _, err := client.Exec(ctx, `UPDATE test SET value = value + 1 WHERE key=$1;`, key); err != nil {
				logrus.WithError(err).Fatal("unexpected error while updating row")
			}
		case deleteEvent:
			idx := rand.Intn(len(existing))
			key := existing[idx]
			if _, err := client.Exec(ctx, `DELETE FROM test WHERE key=$1;`, key); err != nil {
				logrus.WithError(err).Fatal("unexpected error while removing row")
			}
			existing = append(existing[:idx], existing[idx+1:]...)
		default:
			logrus.Fatalf("invalid operation %d", op)
		}
	}

	var finalClusterTimestamp apd.Decimal
	if err := client.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&finalClusterTimestamp); err != nil {
		logrus.WithError(err).Fatal("failed to read initial cluster logical timestamp")
	}
	logrus.Infof("Final cluster timestamp: %s", finalClusterTimestamp.String())

	lock := sync.Mutex{}
	var idx int
	sink := make([][]event, numUpdates+1)
	wg := sync.WaitGroup{}

	into := make([]chan event, numUpdates+1)
	for i := 0; i < numUpdates+1; i++ {
		into[i] = make(chan event)
	}
	launch(ctx, client, &initialClusterTimestamp, &finalClusterTimestamp, &idx, idx, &wg, into, &sink, &lock)

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
	// CRDB does not guarantee ordering between rows, just within them
	for i := range sink {
		sort.Slice(sink[i], func(x, y int) bool {
			return sink[i][x].timestamp.Cmp(sink[i][y].timestamp) < 0
		})
	}

	for i := range sink {
		for j := range sink[i] {
			if sink[i][j].timestamp.Cmp(sink[0][i].timestamp) < 0 {
				logrus.Errorf("changefeed %d saw an event at %s, before the cursor %s", i, sink[i][j].timestamp.String(), sink[0][i].timestamp.String())
			}
		}
	}

	formattedSink := make([][]string, len(sink))
	for i := range sink {
		formattedSink[i] = make([]string, len(sink[i]))
		for j := range sink[i] {
			formattedSink[i][j] = sink[i][j].String()
		}
	}

	reference := formattedSink[0]
	for i, updates := range formattedSink {
		if len(updates) == 0 {
			logrus.Errorf("changefeed %d got no events", i)
			continue
		}
		if actual, expected := len(updates), numUpdates-i; actual != expected {
			logrus.Errorf("changefeed %d got %d events, expected %d", i, actual, expected)
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
			logrus.Errorf("changefeed %d started seeing events at timestamp %q, but the reference watcher never saw that version!", i, updates[0])
			continue
		}
		if startingIndex != i {
			logrus.Errorf("changefeed %d started seeing events at index %d, expected %d", i, startingIndex, i)
		}
		if diff := cmp.Diff(reference[i:], updates); diff != "" {
			logrus.Errorf("changefeed %d got incorrect ordering for events: %v", i, diff)
		}
	}
}

type event struct {
	timestamp *apd.Decimal
	action    string
}

func (e event) String() string {
	return fmt.Sprintf("%s@%s", e.action, e.timestamp)
}

func launch(ctx context.Context, client *pgxpool.Pool, start, end *apd.Decimal, idx *int, i int, wg *sync.WaitGroup, into []chan event, sink *[][]event, lock *sync.Mutex) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		changefeed(ctx, start, end, client, into[i])
		logrus.Infof("Changefeed %d finished.", i)
	}()
	go func() {
		for evt := range into[i] {
			lock.Lock()
			(*sink)[i] = append((*sink)[i], evt)
			if i == 0 {
				*idx++
				launch(ctx, client, evt.timestamp, end, idx, *idx, wg, into, sink, lock)
			}
			lock.Unlock()
		}
	}()
}

func changefeed(ctx context.Context, start, end *apd.Decimal, client *pgxpool.Pool, into chan<- event) {
	options := []string{
		"updated",
		"diff",
		"mvcc_timestamp",
		fmt.Sprintf("cursor='%s'", start.String()),
		"resolved='1s'",
	}
	query := fmt.Sprintf(`EXPERIMENTAL CHANGEFEED FOR test WITH %s;`, strings.Join(options, ","))
	logrus.WithField("sql", query).Info("Exec")
	rows, err := client.Query(ctx, query)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create changefeed")
	}
	defer func() {
		go func() {
			rows.Close()
		}()
	}()
	for rows.Next() {
		if err := rows.Err(); err != nil {
			logrus.WithError(err).Fatal("failed to read changefeed row")
		}

		values := rows.RawValues()
		if len(values) != 3 {
			logrus.Fatalf("expected 3 values in changefeed row, got %d", len(values))
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
			logrus.WithError(err).Fatal("failed to deserialize changefeed row")
		}

		if evt.Resolved != nil {
			if evt.Resolved.Cmp(end) == 1 {
				// we've seen everything we need to see
				logrus.WithField("sql", query).Info("Finished.")
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
			into <- event{
				timestamp: evt.Updated,
				action:    action,
			}
		}
	}
}
