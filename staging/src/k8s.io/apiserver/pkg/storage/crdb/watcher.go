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
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/cockroachdb/apd"
	"github.com/jackc/pgx/v4"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd3/metrics"
	"k8s.io/apiserver/pkg/storage/value"
	utilflowcontrol "k8s.io/apiserver/pkg/util/flowcontrol"

	clientv3 "go.etcd.io/etcd/client/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
)

const (
	// We have set a buffer in order to reduce times of context switches.
	incomingBufSize = 100
	outgoingBufSize = 100
)

// fatalOnDecodeError is used during testing to panic the server if watcher encounters a decoding error
var fatalOnDecodeError = false

// errTestingDecode is the only error that testingDeferOnDecodeError catches during a panic
var errTestingDecode = errors.New("sentinel error only used during testing to indicate watch decoding error")

// testingDeferOnDecodeError is used during testing to recover from a panic caused by errTestingDecode, all other values continue to panic
func testingDeferOnDecodeError() {
	if r := recover(); r != nil && r != errTestingDecode {
		panic(r)
	}
}

func init() {
	// check to see if we are running in a test environment
	TestOnlySetFatalOnDecodeError(true)
	fatalOnDecodeError, _ = strconv.ParseBool(os.Getenv("KUBE_PANIC_WATCH_DECODE_ERROR"))
}

// TestOnlySetFatalOnDecodeError should only be used for cases where decode errors are expected and need to be tested. e.g. conversion webhooks.
func TestOnlySetFatalOnDecodeError(b bool) {
	fatalOnDecodeError = b
}

type watcher struct {
	client      pool
	codec       runtime.Codec
	newFunc     func() runtime.Object
	objectType  string
	versioner   storage.Versioner
	transformer value.Transformer
}

// watchChan implements watch.Interface.
type watchChan struct {
	watcher           *watcher
	key               string
	initialRev        int64
	recursive         bool
	progressNotify    bool
	internalPred      storage.SelectionPredicate
	ctx               context.Context
	cancel            context.CancelFunc
	incomingEventChan chan *event
	resultChan        chan watch.Event
	errChan           chan error

	// HACK: testing watch across multiple prefixes
	cluster *request.Cluster
}

func newWatcher(client pool, codec runtime.Codec, newFunc func() runtime.Object, versioner storage.Versioner, transformer value.Transformer) *watcher {
	res := &watcher{
		client:      client,
		codec:       codec,
		newFunc:     newFunc,
		versioner:   versioner,
		transformer: transformer,
	}
	if newFunc == nil {
		res.objectType = "<unknown>"
	} else {
		res.objectType = reflect.TypeOf(newFunc()).String()
	}
	return res
}

// Watch watches on a key and returns a watch.Interface that transfers relevant notifications.
// If rev is zero, it will return the existing object(s) and then start watching from
// the maximum revision+1 from returned objects.
// If rev is non-zero, it will watch events happened after given revision.
// If recursive is false, it watches on given key.
// If recursive is true, it watches any children and directories under the key, excluding the root key itself.
// pred must be non-nil. Only if pred matches the change, it will be returned.
func (w *watcher) Watch(ctx context.Context, key string, rev int64, recursive bool, cluster *request.Cluster, progressNotify bool, pred storage.SelectionPredicate) (watch.Interface, error) {
	if recursive && !strings.HasSuffix(key, "/") {
		key += "/"
	}
	wc := w.createWatchChan(ctx, key, rev, recursive, cluster, progressNotify, pred)
	go wc.run()

	// For etcd watch we don't have an easy way to answer whether the watch
	// has already caught up. So in the initial version (given that watchcache
	// is by default enabled for all resources but Events), we just deliver
	// the initialization signal immediately. Improving this will be explored
	// in the future.
	utilflowcontrol.WatchInitialized(ctx)

	return wc, nil
}

func (w *watcher) createWatchChan(ctx context.Context, key string, rev int64, recursive bool, cluster *request.Cluster, progressNotify bool, pred storage.SelectionPredicate) *watchChan {
	wc := &watchChan{
		watcher:           w,
		key:               key,
		initialRev:        rev,
		recursive:         recursive,
		progressNotify:    progressNotify,
		internalPred:      pred,
		incomingEventChan: make(chan *event, incomingBufSize),
		resultChan:        make(chan watch.Event, outgoingBufSize),
		errChan:           make(chan error, 1),

		// HACK: assume structure of key is <prefix><cluster>/...
		cluster: cluster,
	}
	if pred.Empty() {
		// The filter doesn't filter out any object.
		wc.internalPred = storage.Everything
	}

	// The etcd server waits until it cannot find a leader for 3 election
	// timeouts to cancel existing streams. 3 is currently a hard coded
	// constant. The election timeout defaults to 1000ms. If the cluster is
	// healthy, when the leader is stopped, the leadership transfer should be
	// smooth. (leader transfers its leadership before stopping). If leader is
	// hard killed, other servers will take an election timeout to realize
	// leader lost and start campaign.
	wc.ctx, wc.cancel = context.WithCancel(clientv3.WithRequireLeader(ctx))
	return wc
}

func (wc *watchChan) run() {
	watchClosedCh := make(chan struct{})
	go wc.startWatching(watchClosedCh)

	var resultChanWG sync.WaitGroup
	resultChanWG.Add(1)
	go wc.processEvent(&resultChanWG)

	select {
	case err := <-wc.errChan:
		if err == context.Canceled {
			break
		}
		errResult := transformErrorToEvent(err)
		if errResult != nil {
			// error result is guaranteed to be received by user before closing ResultChan.
			select {
			case wc.resultChan <- *errResult:
			case <-wc.ctx.Done(): // user has given up all results
			}
		}
	case <-watchClosedCh:
	case <-wc.ctx.Done(): // user cancel
	}

	// We use wc.ctx to reap all goroutines. Under whatever condition, we should stop them all.
	// It's fine to double cancel.
	wc.cancel()

	// we need to wait until resultChan wouldn't be used anymore
	resultChanWG.Wait()
	close(wc.resultChan)
}

func (wc *watchChan) Stop() {
	wc.cancel()
}

func (wc *watchChan) ResultChan() <-chan watch.Event {
	return wc.resultChan
}

// sync tries to retrieve existing data and send them to process.
// The revision to watch will be set to the revision in response.
// All events sent will have isCreated=true
func (wc *watchChan) sync() error {
	// TODO: with the enterprise version of CRDB this whole sync() dance becomes
	// CREATE CHANGEFEED FOR ... WITH cursor=<>,initial_scan
	requestInfo, haveRequestInfo := request.RequestInfoFrom(wc.ctx)
	if !haveRequestInfo {
		return storage.NewInternalError("no request info metadata found")
	}

	var clusterTimestamp apd.Decimal
	var events []*event
	if err := wc.watcher.client.BeginTxFunc(wc.ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if !wc.recursive {
			if wc.cluster == nil || wc.cluster.Wildcard || wc.cluster.Name == "" {
				return fmt.Errorf("invalid cluster, need a single selected name: %#v", wc.cluster)
			}
			// we are watching one key only
			var data []byte
			var hybridLogicalTimestamp apd.Decimal
			err := tx.QueryRow(wc.ctx, `SELECT value, crdb_internal_mvcc_timestamp FROM k8s WHERE key=$1 AND cluster=$2;`, wc.key, wc.cluster.Name).Scan(&data, &hybridLogicalTimestamp)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("failed to read key when starting watch: %w", err)
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				objectResourceVersion, err := toResourceVersion(&hybridLogicalTimestamp)
				if err != nil {
					return fmt.Errorf("failed to parse resourceVersion of key when starting watch: %w", err)
				}
				events = append(events, parseData(wc.key, data, objectResourceVersion))
			}
		} else {
			// we are watching a list
			clauses, args := whereClause(2, wc.cluster, requestInfo)
			args = append([]interface{}{wc.key}, args...)
			query := fmt.Sprintf(`SELECT key, value, crdb_internal_mvcc_timestamp FROM k8s WHERE key LIKE '%s%%' AND key >= $1%s ORDER BY key;`, wc.key, clauses)
			rows, err := tx.Query(wc.ctx, query, args...)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("failed to query rows when starting watch: %w", err)
			}
			defer func() {
				rows.Close()
			}()

			for rows.Next() {
				if err := rows.Err(); err != nil {
					return fmt.Errorf("failed to query row when starting watch: %w", err)
				}

				var key string
				var data []byte
				var hybridLogicalTimestamp apd.Decimal
				if err := rows.Scan(&key, &data, &hybridLogicalTimestamp); err != nil {
					return fmt.Errorf("failed to read row when starting watch: %w", err)
				}

				objectResourceVersion, err := toResourceVersion(&hybridLogicalTimestamp)
				if err != nil {
					return fmt.Errorf("failed to parse resourceVersion of row when starting watch: %w", err)
				}
				events = append(events, parseData(key, data, objectResourceVersion))
			}
		}

		timestampErr := tx.QueryRow(wc.ctx, `SELECT cluster_logical_timestamp();`).Scan(&clusterTimestamp)
		if timestampErr != nil {
			return fmt.Errorf("failed to read cluster timestamp: %w", timestampErr)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to sync data when starting watch: %v", err)
	}

	latestResourceVersion, err := toResourceVersion(&clusterTimestamp)
	if err != nil {
		return err
	}

	wc.initialRev = latestResourceVersion
	for _, evt := range events {
		wc.sendEvent(evt)
	}
	return nil
}

// logWatchChannelErr checks whether the error is about mvcc revision compaction which is regarded as warning
func logWatchChannelErr(err error) {
	if !isGarbageCollectionError(err) {
		klog.Errorf("watch chan error: %v", err)
	} else {
		klog.Warningf("watch chan error: %v", err)
	}
}

type changefeedEvent struct {
	MVCCTimestamp *apd.Decimal `json:"mvcc_timestamp,omitempty"`
	Updated       *apd.Decimal `json:"Updated,omitempty"`
	Resolved      *apd.Decimal `json:"resolved,omitempty"`

	After *row `json:"after,omitempty"`
}

type row struct {
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Cluster string `json:"cluster,omitempty"`
}

// startWatching does:
// - get current objects if initialRev=0; set initialRev to current rev
// - watch on given key and send events to process.
func (wc *watchChan) startWatching(watchClosedCh chan struct{}) {
	if wc.initialRev == 0 {
		if err := wc.sync(); err != nil {
			err = fmt.Errorf("failed to sync with latest state: %w", err)
			klog.Error(err)
			wc.sendError(err)
			return
		}
	}
	providedTimestamp, err := toHybridLogicalClock(wc.initialRev)
	if err != nil {
		err = fmt.Errorf("failed to parse initial revision: %w", err)
		klog.Error(err)
		wc.sendError(err)
		return
	}
	var initialWatchTimestamp apd.Decimal
	condition, err := apd.BaseContext.Add(&initialWatchTimestamp, providedTimestamp, apd.New(1, 0))
	if err != nil {
		err = fmt.Errorf("failed to determine initial watch revision: %v", err)
		klog.Error(err)
		wc.sendError(err)
		return
	}
	if _, err := condition.GoError(apd.DefaultTraps); err != nil {
		err = fmt.Errorf("failed to determine initial watch revision: %v", err)
		klog.Error(err)
		wc.sendError(err)
		return
	}
	options := []string{
		"updated",
		"mvcc_timestamp",
		fmt.Sprintf("cursor='%s'", initialWatchTimestamp.String()),
	}
	if wc.progressNotify {
		options = append(options, "resolved='1s'") // TODO: this is a server setting in etcd, but a client one for us
	}
	query := fmt.Sprintf(`EXPERIMENTAL CHANGEFEED FOR k8s WITH %s;`, strings.Join(options, ","))
	// TODO: the SQL logger only prints the command when it finishes, so we never see it ...
	klog.InfoS("Exec", "sql", query)
	rows, err := wc.watcher.client.Query(wc.ctx, query)
	if err != nil {
		// If there is an error on server (e.g. compaction), the channel will return it before closed.
		err = fmt.Errorf("failed to create changefeed: %w", err)
		logWatchChannelErr(err)
		wc.sendError(err)
		return
	}
	defer func() {
		rows.Close()
	}()
	for rows.Next() {
		if err := rows.Err(); err != nil {
			err = fmt.Errorf("failed to read changefeed row: %w", err)
			logWatchChannelErr(err)
			wc.sendError(err)
			return
		}

		values := rows.RawValues()
		if len(values) != 3 {
			err := fmt.Errorf("expected 3 values in changefeed row, got %d", len(values))
			logWatchChannelErr(err)
			wc.sendError(err)
			return
		}
		// values upacks into (tableName, primaryKey, rowData)
		// tableName = name
		// primaryKey = ["array", "of", "values"]
		// rowData = {...}
		primaryKeysRaw, data := values[1], values[2]

		var key, cluster string
		if len(primaryKeysRaw) != 0 {
			// this is insanity but not sure what the best way to parse this is ... ?
			type wrappedPrimaryKey struct {
				Key []string `json:"key"`
			}
			jsonPrimaryKey := fmt.Sprintf(`{"key":%s}`, string(primaryKeysRaw))
			var wrappedJsonPrimaryKey wrappedPrimaryKey
			if err := json.Unmarshal([]byte(jsonPrimaryKey), &wrappedJsonPrimaryKey); err != nil {
				err = fmt.Errorf("failed to deserialize changefeed primary key: %w", err)
				logWatchChannelErr(err)
				wc.sendError(err)
				return
			}
			primaryKeys := wrappedJsonPrimaryKey.Key
			if len(primaryKeys) != 2 {
				// this should be (key, cluster) as defined in the CREATE TABLE
				err := fmt.Errorf("expected 2 values in changefeed row primary key, got %d", len(primaryKeys))
				logWatchChannelErr(err)
				wc.sendError(err)
				return
			}
			key, cluster = primaryKeys[0], primaryKeys[1]
		}

		var evt changefeedEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			err = fmt.Errorf("failed to deserialize changefeed row: %w", err)
			logWatchChannelErr(err)
			wc.sendError(err)
			return
		}

		// sanity check
		if evt.After != nil {
			if evt.After.Key != key {
				err := fmt.Errorf("expected key in primary key (%s) to match key in row (%s)", key, evt.After.Key)
				logWatchChannelErr(err)
				wc.sendError(err)
				return
			}
			if evt.After.Cluster != cluster {
				err := fmt.Errorf("expected cluster in primary key (%s) to match cluster in row (%s)", cluster, evt.After.Cluster)
				logWatchChannelErr(err)
				wc.sendError(err)
				return
			}
		}

		if evt.Resolved != nil {
			resourceVersion, err := toResourceVersion(evt.Resolved)
			if err != nil {
				logWatchChannelErr(err)
				wc.sendError(err)
				return
			}
			wc.sendEvent(progressNotifyEvent(resourceVersion))
			metrics.RecordEtcdBookmark(wc.watcher.objectType)
			continue
		}

		keyFilter := func(incomingKey string) bool {
			if wc.recursive {
				return strings.HasPrefix(incomingKey, wc.key)
			}
			return incomingKey == wc.key
		}
		clusterFilter := func(incomingCluster string) bool {
			if wc.cluster != nil && !wc.cluster.Wildcard && wc.cluster.Name != "" {
				return incomingCluster == wc.cluster.Name
			}
			return true
		}

		if !keyFilter(key) || !clusterFilter(cluster) {
			// this watch should not emit this event
			continue
		}

		if evt.Updated != nil {
			resourceVersion, err := toResourceVersion(evt.Updated)
			if err != nil {
				logWatchChannelErr(err)
				wc.sendError(err)
				return
			}

			// TODO: this is likely better done with a caching layer or something, but the footprint will be enormous
			// fetch the key at a previous RV - if it existed, we will have a previous version to show the client
			// who is watching; if it did not, we know this is a "create" event.
			// NOTE: for some reason k8s watch sends the current RV for the old object ... ?
			objectTimestamp, err := toHybridLogicalClock(resourceVersion)
			if err != nil {
				err = fmt.Errorf("failed to determine hybrid logical clock when handling watch event: %w", err)
				logWatchChannelErr(err)
				wc.sendError(err)
				return
			}
			var previousTimestamp apd.Decimal
			condition, err := apd.BaseContext.Sub(&previousTimestamp, objectTimestamp, apd.New(1, 0))
			if err != nil {
				err = fmt.Errorf("failed to determine previous object revision: %v", err)
				klog.Error(err)
				wc.sendError(err)
				return
			}
			if _, err := condition.GoError(apd.DefaultTraps); err != nil {
				err = fmt.Errorf("failed to determine previous object revision: %v", err)
				klog.Error(err)
				wc.sendError(err)
				return
			}

			var previousData []byte
			err = wc.watcher.client.QueryRow(wc.ctx, fmt.Sprintf(`SELECT value FROM k8s AS OF SYSTEM TIME %s WHERE key=$1 AND cluster=$2;`, previousTimestamp.String()), key, cluster).Scan(&previousData)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				err = fmt.Errorf("failed to read previous key when handling watch event: %w", err)
				logWatchChannelErr(err)
				wc.sendError(err)
				return
			}

			parsedEvent, err := parseEvent(key, &evt, previousData, resourceVersion)
			if err != nil {
				logWatchChannelErr(err)
				wc.sendError(err)
				return
			}
			wc.sendEvent(parsedEvent)
		}
	}
	// When we come to this point, it's only possible that client side ends the watch.
	// e.g. cancel the context, close the client.
	// If this watch chan is broken and context isn't cancelled, other goroutines will still hang.
	// We should notify the main thread that this goroutine has exited.
	close(watchClosedCh)
}

// processEvent processes events from etcd watcher and sends results to resultChan.
func (wc *watchChan) processEvent(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case e := <-wc.incomingEventChan:
			res := wc.transform(e)
			if res == nil {
				continue
			}
			if len(wc.resultChan) == outgoingBufSize {
				klog.V(3).InfoS("Fast watcher, slow processing. Probably caused by slow dispatching events to watchers", "outgoingEvents", outgoingBufSize, "objectType", wc.watcher.objectType)
			}
			// If user couldn't receive results fast enough, we also block incoming events from watcher.
			// Because storing events in local will cause more memory usage.
			// The worst case would be closing the fast watcher.
			select {
			case wc.resultChan <- *res:
			case <-wc.ctx.Done():
				return
			}
		case <-wc.ctx.Done():
			return
		}
	}
}

func (wc *watchChan) filter(obj runtime.Object) bool {
	if wc.internalPred.Empty() {
		return true
	}
	matched, err := wc.internalPred.Matches(obj)
	return err == nil && matched
}

func (wc *watchChan) acceptAll() bool {
	return wc.internalPred.Empty()
}

// transform transforms an event into a result for user if not filtered.
func (wc *watchChan) transform(e *event) (res *watch.Event) {
	curObj, oldObj, err := wc.prepareObjs(e)
	if err != nil {
		klog.Errorf("failed to prepare current and previous objects: %v", err)
		wc.sendError(err)
		return nil
	}

	switch {
	case e.isProgressNotify:
		if wc.watcher.newFunc == nil {
			return nil
		}
		object := wc.watcher.newFunc()
		if err := wc.watcher.versioner.UpdateObject(object, uint64(e.rev)); err != nil {
			klog.Errorf("failed to propagate object version: %v", err)
			return nil
		}
		res = &watch.Event{
			Type:   watch.Bookmark,
			Object: object,
		}
	case e.isDeleted:
		if !wc.filter(oldObj) {
			return nil
		}
		res = &watch.Event{
			Type:   watch.Deleted,
			Object: oldObj,
		}
	case e.isCreated:
		if !wc.filter(curObj) {
			return nil
		}
		res = &watch.Event{
			Type:   watch.Added,
			Object: curObj,
		}
	default:
		if wc.acceptAll() {
			res = &watch.Event{
				Type:   watch.Modified,
				Object: curObj,
			}
			return res
		}
		curObjPasses := wc.filter(curObj)
		oldObjPasses := wc.filter(oldObj)
		switch {
		case curObjPasses && oldObjPasses:
			res = &watch.Event{
				Type:   watch.Modified,
				Object: curObj,
			}
		case curObjPasses && !oldObjPasses:
			res = &watch.Event{
				Type:   watch.Added,
				Object: curObj,
			}
		case !curObjPasses && oldObjPasses:
			res = &watch.Event{
				Type:   watch.Deleted,
				Object: oldObj,
			}
		}
	}
	return res
}

func transformErrorToEvent(err error) *watch.Event {
	err = interpretWatchError(err)
	if _, ok := err.(apierrors.APIStatus); !ok {
		err = apierrors.NewInternalError(err)
	}
	status := err.(apierrors.APIStatus).Status()
	return &watch.Event{
		Type:   watch.Error,
		Object: &status,
	}
}

func (wc *watchChan) sendError(err error) {
	select {
	case wc.errChan <- err:
	case <-wc.ctx.Done():
	}
}

func (wc *watchChan) sendEvent(e *event) {
	if len(wc.incomingEventChan) == incomingBufSize {
		klog.V(3).InfoS("Fast watcher, slow processing. Probably caused by slow decoding, user not receiving fast, or other processing logic", "incomingEvents", incomingBufSize, "objectType", wc.watcher.objectType)
	}
	select {
	case wc.incomingEventChan <- e:
	case <-wc.ctx.Done():
	}
}

func (wc *watchChan) prepareObjs(e *event) (curObj runtime.Object, oldObj runtime.Object, err error) {
	if e.isProgressNotify {
		// progressNotify events doesn't contain neither current nor previous object version,
		return nil, nil, nil
	}

	if !e.isDeleted {
		data, _, err := wc.watcher.transformer.TransformFromStorage(e.value, authenticatedDataString(e.key))
		if err != nil {
			return nil, nil, err
		}
		curObj, err = decodeObj(wc.watcher.codec, wc.watcher.versioner, data, e.rev)
		if err != nil {
			return nil, nil, err
		}
		clusterName := wc.cluster.Name
		if wc.cluster.Wildcard {
			sub := strings.TrimPrefix(e.key, wc.key)
			if i := strings.Index(sub, "/"); i != -1 {
				sub = sub[:i]
			}
			clusterName = sub
		}
		// HACK: in order to support CRD tenancy, the clusterName, which is extracted from the object etcd key,
		// should be set on the decoded object.
		// This is done here since we want to set the logical cluster the object is part of,
		// without storing the clusterName inside the etcd object itself (as it has been until now).
		// The etcd key is ultimately the only thing that links us to a cluster
		if clusterName != "" {
			if s, ok := curObj.(metav1.ObjectMetaAccessor); ok {
				s.GetObjectMeta().SetClusterName(clusterName)
			} else if s, ok := curObj.(metav1.Object); ok {
				s.SetClusterName(clusterName)
			} else if s, ok := curObj.(*unstructured.Unstructured); ok {
				s.SetClusterName(clusterName)
			} else {
				klog.Warningf("Could not set ClusterName %s in prepareObjs on object: %T", clusterName, curObj)
			}
		} else {
			klog.Errorf("Cluster should not be unknown")
		}
	}
	// We need to decode prevValue, only if this is deletion event or
	// the underlying filter doesn't accept all objects (otherwise we
	// know that the filter for previous object will return true and
	// we need the object only to compute whether it was filtered out
	// before).
	if len(e.prevValue) > 0 && (e.isDeleted || !wc.acceptAll()) {
		data, _, err := wc.watcher.transformer.TransformFromStorage(e.prevValue, authenticatedDataString(e.key))
		if err != nil {
			return nil, nil, err
		}
		// Note that this sends the *old* object with the etcd revision for the time at
		// which it gets deleted.
		oldObj, err = decodeObj(wc.watcher.codec, wc.watcher.versioner, data, e.rev)
		if err != nil {
			return nil, nil, err
		}
		clusterName := wc.cluster.Name
		if wc.cluster.Wildcard {
			sub := strings.TrimPrefix(e.key, wc.key)
			if i := strings.Index(sub, "/"); i != -1 {
				sub = sub[:i]
			}
			clusterName = sub
		}
		// HACK: in order to support CRD tenancy, the clusterName, which is extracted from the object etcd key,
		// should be set on the decoded object.
		// This is done here since we want to set the logical cluster the object is part of,
		// without storing the clusterName inside the etcd object itself (as it has been until now).
		// The etcd key is ultimately the only thing that links us to a cluster
		if clusterName != "" {
			if s, ok := oldObj.(metav1.ObjectMetaAccessor); ok {
				s.GetObjectMeta().SetClusterName(clusterName)
			} else if s, ok := oldObj.(metav1.Object); ok {
				s.SetClusterName(clusterName)
			} else if s, ok := oldObj.(*unstructured.Unstructured); ok {
				s.SetClusterName(clusterName)
			} else {
				klog.Warningf("Could not set ClusterName %s in prepareObjs on object: %T", clusterName, curObj)
			}
		} else {
			klog.Errorf("Cluster should not be unknown")
		}
	}
	return curObj, oldObj, nil
}

func decodeObj(codec runtime.Codec, versioner storage.Versioner, data []byte, rev int64) (_ runtime.Object, err error) {
	obj, err := runtime.Decode(codec, []byte(data))
	if err != nil {
		if fatalOnDecodeError {
			// catch watch decode error iff we caused it on
			// purpose during a unit test
			defer testingDeferOnDecodeError()
			// we are running in a test environment and thus an
			// error here is due to a coder mistake if the defer
			// does not catch it
			panic(err)
		}
		return nil, err
	}
	// ensure resource version is set on the object we load from etcd
	if err := versioner.UpdateObject(obj, uint64(rev)); err != nil {
		return nil, fmt.Errorf("failure to version api object (%d) %#v: %v", rev, obj, err)
	}
	return obj, nil
}
