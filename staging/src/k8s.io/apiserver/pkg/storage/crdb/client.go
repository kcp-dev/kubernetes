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
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/cockroachdb/apd"
	"github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/google/uuid"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"

	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/generic"
	"k8s.io/klog/v2"
)

// pool holds the methods we need, so we can mock this out in tests
type pool interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

func NewClient(ctx context.Context, enableCaching bool, c pool) *client {
	var cache *changefeedCache
	if enableCaching {
		cache = initialize(ctx, c)
	}
	return &client{
		pool:            c,
		changefeedCache: cache,
	}
}

type client struct {
	pool
	*changefeedCache
}

func (c *client) Get(ctx context.Context, key string) (response *generic.Response, err error) {
	// NOTE: the k8s GET semantics are weird - when passing a resourceVersion in a GET parameter,
	// you are only ever going to get something that is no older than the resourceVersion, and the
	// check is done against the logical time *of the read*, not of the last write to the data...
	// Since we always need to read the current timestamp to make this check (even when the row
	// being fetched is missing), we need to use a transaction here instead of a single SELECT.
	var data []byte
	var hybridLogicalTimestamp apd.Decimal
	var clusterTimestamp apd.Decimal
	var missing bool
	if err := crdbpgx.ExecuteTx(ctx, c.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&clusterTimestamp); err != nil {
			return err
		}
		return c.QueryRow(ctx, `SELECT value, crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1;`, key).Scan(&data, &hybridLogicalTimestamp)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// this is not necessarily an error for our caller
			missing = true
		} else {
			return nil, storage.NewInternalError(err.Error())
		}
	}

	latestResourceVersion, err := toResourceVersion(&clusterTimestamp)
	if err != nil {
		return nil, err
	}
	resp := &generic.Response{Header: &generic.ResponseHeader{Revision: latestResourceVersion}}

	if !missing {
		objectResourceVersion, err := toResourceVersion(&hybridLogicalTimestamp)
		if err != nil {
			return nil, err
		}
		resp.Kvs = []*generic.KeyValue{
			{Key: []byte(key), Value: data, ModRevision: objectResourceVersion},
		}
	}
	return resp, nil
}

func (c *client) Create(ctx context.Context, key string, value []byte, ttl uint64) (succeeded bool, response *generic.Response, err error) {
	if ttl != 0 {
		// TODO: we could try the workarounds? https://github.com/cockroachdb/cockroach/issues/20239
		klog.Error("crdb does not support row-level TTLs")
	}
	thisUuid := uuid.New()
	// we can't use INSERT ... RETURNING crdb_internal_mvcc_timestamp
	// so we need to use a transaction here to ensure we do not race.
	// Reading the crdb_internal_mvcc_timestamp will give us the provisional
	// commit timestamp, which is not guaranteed to be correct. Instead,
	// we read the cluster_logical_timestamp(), which locks us into that time.
	// https://github.com/cockroachdb/cockroach/issues/58046
	var hybridLogicalTimestamp apd.Decimal
	if err := crdbpgx.ExecuteTx(ctx, c.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO k8s (key, value) VALUES ($1, $2);`, key, value); err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `INSERT INTO k8s_causality_hack (id) VALUES ($1);`, thisUuid)
		return err
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return false, nil, storage.NewKeyExistsError(key, 0)
			}
		}
		return false, nil, storage.NewInternalError(err.Error())
	}

	if err := c.QueryRow(ctx, `SELECT crdb_internal_mvcc_timestamp FROM k8s_causality_hack WHERE id=$1`, thisUuid).Scan(&hybridLogicalTimestamp); err != nil {
		return false, nil, storage.NewInternalError(err.Error())
	}

	resourceVersion, err := toResourceVersion(&hybridLogicalTimestamp)
	if err != nil {
		return false, nil, storage.NewInternalError(err.Error())
	}

	return true, &generic.Response{Header: &generic.ResponseHeader{Revision: resourceVersion}}, nil
}

func (c *client) ConditionalDelete(ctx context.Context, key string, previousRevision int64) (succeeded bool, response *generic.Response, err error) {
	previousHybridLogicalTimestamp, err := toHybridLogicalClock(previousRevision)
	if err != nil {
		return false, nil, storage.NewInternalError(err.Error())
	}

	var retry, missing bool
	var hybridLogicalTimestamp apd.Decimal
	var data []byte
	if err := crdbpgx.ExecuteTx(ctx, c.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM k8s WHERE key=$1 AND crdb_internal_mvcc_timestamp=$2;`, key, previousHybridLogicalTimestamp)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			retry = true
			// we didn't delete anything, either because the value we used for our precondition was too old,
			// or because someone raced with us to delete the thing - so, check to see if there's some updated
			// value we can use for preconditions next time around
			return tx.QueryRow(ctx, `SELECT value, crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1;`, key).Scan(&data, &hybridLogicalTimestamp)
		}
		return nil
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// this is not necessarily an error for our caller
			missing = true
		} else {
			return false, nil, storage.NewInternalError(err.Error())
		}
	}
	resp := &generic.Response{}

	if retry && !missing {
		objectResourceVersion, rvErr := toResourceVersion(&hybridLogicalTimestamp)
		if rvErr != nil {
			return false, nil, rvErr
		}
		resp.Kvs = []*generic.KeyValue{
			{Key: []byte(key), Value: data, ModRevision: objectResourceVersion},
		}
	}
	return !retry, resp, nil
}

func (c *client) ConditionalUpdate(ctx context.Context, key string, value []byte, previousRevision int64, ttl uint64) (succeeded bool, response *generic.Response, err error) {
	if ttl != 0 {
		// TODO: we could try the workarounds? https://github.com/cockroachdb/cockroach/issues/20239
		klog.Error("crdb does not support row-level TTLs")
	}

	previousHybridLogicalTimestamp, err := toHybridLogicalClock(previousRevision)
	if err != nil {
		return false, nil, storage.NewInternalError(err.Error())
	}

	thisUuid := uuid.New()
	var retry, missing bool
	var hybridLogicalTimestamp apd.Decimal
	var data []byte
	if err := crdbpgx.ExecuteTx(ctx, c.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// TODO: there was some reason I did not use UPSERT here - was it the WHERE clause? Reconsider: https://www.cockroachlabs.com/docs/v21.2/performance-best-practices-overview#use-upsert-instead-of-insert-on-conflict-on-tables-with-no-secondary-indexes
		tag, execErr := tx.Exec(ctx, `INSERT INTO k8s (key, value) 
											VALUES ($1, $2) 
											ON CONFLICT (key) 
											DO UPDATE SET value=excluded.value
											WHERE crdb_internal_mvcc_timestamp=$3;`,
			key, value, previousHybridLogicalTimestamp,
		)
		if execErr != nil {
			return execErr
		}
		if tag.RowsAffected() == 0 {
			retry = true
			// we didn't update anything, either because the value we used for our precondition was too old,
			// or because someone raced with us to delete the thing - so, check to see if there's some updated
			// value we can use for preconditions next time around
			return tx.QueryRow(ctx, `SELECT value, crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1;`, key).Scan(&data, &hybridLogicalTimestamp)
		}
		// read the updated timestamp so we know the new resourceVersion
		_, err = tx.Exec(ctx, `INSERT INTO k8s_causality_hack (id) VALUES ($1);`, thisUuid)
		return err
		//return tx.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&hybridLogicalTimestamp)
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// this is not necessarily an error for our caller
			missing = true
		} else {
			return false, nil, storage.NewInternalError(err.Error())
		}
	}
	if !retry {
		if err := c.QueryRow(ctx, `SELECT crdb_internal_mvcc_timestamp FROM k8s_causality_hack WHERE id=$1`, thisUuid).Scan(&hybridLogicalTimestamp); err != nil {
			return false, nil, storage.NewInternalError(err.Error())
		}
	}
	updatedResourceVersion, rvErr := toResourceVersion(&hybridLogicalTimestamp)
	if rvErr != nil {
		return false, nil, rvErr
	}
	resp := &generic.Response{Header: &generic.ResponseHeader{Revision: updatedResourceVersion}}

	if retry && !missing {
		objectResourceVersion, rvErr := toResourceVersion(&hybridLogicalTimestamp)
		if rvErr != nil {
			return false, nil, rvErr
		}
		resp.Kvs = []*generic.KeyValue{
			{Key: []byte(key), Value: data, ModRevision: objectResourceVersion},
		}
	}
	return !retry, resp, nil
}

func (c *client) Count(ctx context.Context, key string) (count int64, err error) {
	if err := c.QueryRow(context.Background(), `SELECT COUNT(*) FROM k8s WHERE key LIKE $1;`, fmt.Sprintf("%s%%", key)).Scan(&count); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, storage.NewKeyNotFoundError(key, 0)
		}
		return 0, storage.NewInternalError(err.Error())
	}
	return count, nil
}

func (c *client) List(ctx context.Context, key, prefix string, recursive, paging bool, limit, revision int64) (response *generic.Response, err error) {
	var revisionClause, limitClause string
	if revision != 0 {
		providedTimestamp, err := toHybridLogicalClock(revision)
		if err != nil {
			return nil, err
		}
		// TODO: can we do something better here for the time? this is an SQL injection attack vector, but using a var makes:
		// ERROR: AS OF SYSTEM TIME: only constant expressions, with_min_timestamp, with_max_staleness, or follower_read_timestamp are allowed (SQLSTATE XXUUU)
		// https://github.com/cockroachdb/cockroach/issues/30955
		revisionClause = fmt.Sprintf(`AS OF SYSTEM TIME %s `, providedTimestamp.String())
	}
	if limit > 0 {
		limitClause = fmt.Sprintf(` LIMIT %d`, limit)
	}

	var finalKey, lastKey []byte
	var remaining int64
	var clusterTimestamp apd.Decimal
	var kvs []*generic.KeyValue
	// as always, the resourceVersion parameter the client passes in is compared to the
	// latest possible system clock, not the last write to the data, so we need to use a
	// transaction here to read that ...
	// TODO: we're hacking the pgx.TxOptions here for `AS OF SYSTEM TIME` ... but there's no other support for it?
	if err := crdbpgx.ExecuteTx(ctx, c.pool, pgx.TxOptions{DeferrableMode: pgx.TxDeferrableMode(revisionClause)}, func(tx pgx.Tx) error {
		// we need to read the cluster timestamp even if the data read fails with a NotFound
		if err := tx.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&clusterTimestamp); err != nil {
			return err
		}

		// TODO: perhaps just for tests, we need WHERE key LIKE prefix% since those tests expect real prefixing ... but that is not ideal for production
		if err := tx.QueryRow(ctx, `SELECT COUNT(*), MAX(key) FROM k8s WHERE key LIKE $1 AND key >= $2;`, fmt.Sprintf("%s%%", prefix), key).Scan(&remaining, &finalKey); err != nil {
			return err
		}

		query := fmt.Sprintf(`SELECT key, value, crdb_internal_mvcc_timestamp FROM k8s WHERE key LIKE $1 AND key >= $2 ORDER BY key%s;`, limitClause)
		rows, err := tx.Query(ctx, query, fmt.Sprintf("%s%%", prefix), key)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		defer func() {
			rows.Close()
		}()

		for rows.Next() {
			if err := rows.Err(); err != nil {
				return err
			}

			var data []byte
			var hybridLogicalTimestamp apd.Decimal
			if err := rows.Scan(&lastKey, &data, &hybridLogicalTimestamp); err != nil {
				return err
			}

			objectResourceVersion, err := toResourceVersion(&hybridLogicalTimestamp)
			if err != nil {
				return err
			}

			kvs = append(kvs, &generic.KeyValue{
				Key:         lastKey,
				Value:       data,
				ModRevision: objectResourceVersion,
			})
		}

		return nil
	}); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, transformError(err)
		}
	}

	latestResourceVersion, err := toResourceVersion(&clusterTimestamp)
	if err != nil {
		return nil, err
	}

	return &generic.Response{
		Header: &generic.ResponseHeader{Revision: latestResourceVersion},
		Kvs:    kvs,
		Count:  remaining,
		More:   !bytes.Equal(lastKey, finalKey),
	}, nil
}

func isGarbageCollectionError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "XXUUU" && strings.Contains(pgErr.Message, "must be after replica GC threshold") {
			return true
		}
	}
	return false
}

func transformError(err error) error {
	if isGarbageCollectionError(err) {
		return generic.ErrCompacted
	}
	return err
}

func (c *client) Watch(ctx context.Context, key string, recursive, progressNotify bool, revision int64) <-chan *generic.WatchResponse {
	if c.changefeedCache != nil {
		return watchFromCache(ctx, c.changefeedCache, key, recursive, progressNotify, revision)
	}
	return watchFromChangefeed(ctx, c.pool, key, recursive, progressNotify, revision)
}

func watchFromCache(ctx context.Context, cache *changefeedCache, key string, recursive, progressNotify bool, revision int64) <-chan *generic.WatchResponse {
	buffer := make(chan *generic.WatchResponse, 10)
	events := make(chan *generic.WatchResponse)
	go func() {
		initialWatchTimestamp, err := toHybridLogicalClock(revision)
		if err != nil {
			err = fmt.Errorf("failed to parse initial revision: %w", err)
			klog.Error(err)
			close(buffer)
			events <- &generic.WatchResponse{Error: err}
			return
		}
		cache.subscribeAfter(ctx, initialWatchTimestamp, buffer)
	}()

	// the changefeed cache will send us all events, so we need to filter for
	// keys the caller is actually interested in, similarly with progress notifications
	go filter(ctx, buffer, events, key, recursive, progressNotify)

	return events
}

func watchFromChangefeed(ctx context.Context, client pool, key string, recursive, progressNotify bool, revision int64) <-chan *generic.WatchResponse {
	buffer := make(chan *watchEvent)
	reduced := make(chan *generic.WatchResponse)
	released := make(chan *generic.WatchResponse)
	events := make(chan *generic.WatchResponse)
	go func() {
		initialWatchTimestamp := &apd.Decimal{}
		var err error
		if revision == 0 {
			err = client.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(initialWatchTimestamp)
		} else {
			initialWatchTimestamp, err = toHybridLogicalClock(revision)
		}
		if err != nil {
			err = fmt.Errorf("failed to parse initial revision: %w", err)
			klog.Error(err)
			close(buffer)
			events <- &generic.WatchResponse{Error: err}
			return
		}
		consumeChangefeed(ctx, client, initialWatchTimestamp, buffer)
	}()

	// adapt the internal watch events to the generic ones
	go reduce(ctx, buffer, reduced)
	// reorder events as they come in so downstream consumers are always in order
	go release(ctx, reduced, released)
	// filter the watch events for those we are interested in
	go filter(ctx, released, events, key, recursive, progressNotify)

	return events
}

func reduce(ctx context.Context, source <-chan *watchEvent, sink chan<- *generic.WatchResponse) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-source:
			select {
			case <-ctx.Done():
			case sink <- evt.WatchResponse:
			}
		}
	}
}

func release(ctx context.Context, source <-chan *generic.WatchResponse, sink chan<- *generic.WatchResponse) {
	var buffer []*generic.WatchResponse
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-source:
			if evt.ProgressNotify {
				for _, item := range buffer {
					select {
					case <-ctx.Done():
					case sink <- item:
					}
				}
				select {
				case <-ctx.Done():
				case sink <- evt:
				}
				buffer = []*generic.WatchResponse{}
				continue
			} else {
				i := sort.Search(len(buffer), func(i int) bool { return buffer[i].Header.Revision > evt.Header.Revision })
				if i < len(buffer) && buffer[i].Header.Revision == evt.Header.Revision {
					// this is a duplicate event, ignore it
					return
				}
				// append into buffer, sorted
				if i == len(buffer) {
					buffer = append(buffer, evt)
				} else {
					buffer = append(buffer[:i], append([]*generic.WatchResponse{evt}, buffer[i:]...)...)
				}
			}
		}
	}
}

func filter(ctx context.Context, source <-chan *generic.WatchResponse, sink chan<- *generic.WatchResponse, key string, recursive, progressNotify bool) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-source:
			if evt.ProgressNotify {
				if progressNotify {
					select {
					case <-ctx.Done():
					case sink <- evt:
					}
				}
				continue
			}

			keyFilter := func(incomingKey string) bool {
				if recursive {
					return strings.HasPrefix(incomingKey, key)
				}
				return incomingKey == key
			}
			var incomingKey string
			if evt.Events[0].Kv != nil {
				incomingKey = string(evt.Events[0].Kv.Key)
			} else if evt.Events[0].PrevKv != nil {
				incomingKey = string(evt.Events[0].PrevKv.Key)
			}

			if !keyFilter(incomingKey) {
				// this watch should not emit this event
				continue
			}
			select {
			case <-ctx.Done():
			case sink <- evt:
			}
		}
	}
}

var _ generic.Client = &client{}
