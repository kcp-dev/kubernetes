package crdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cockroachdb/apd"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgx/v4"
)

func newEvent(i int) *event {
	return &event{
		key:    "test",
		rawRev: apd.New(int64(i), 0),
		eType:  eventTypeProgressNotify,
	}
}

func TestSkipListAppend(t *testing.T) {
	sl := &skipList{
		RWMutex: &sync.RWMutex{},
		heads:   map[int]*skipListNode{},
		tails:   map[int]*skipListNode{},
		tail:    0,
	}
	for i := 0; i < 9; i++ {
		sl.append(newEvent(i))
	}
	// we expect the following pointers:
	// 0: 0 -> 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> nil
	// 1: 0 ------> 2 ------> 4 ------> 6 ------> 8 -> nil
	// 2: 0 ----------------> 4 ----------------> 8 -> nil
	// 3: 0 ------------------------------------> 8 -> nil
	for level, items := range [][]int64{
		{0, 1, 2, 3, 4, 5, 6, 7, 8},
		{0, 2, 4, 6, 8},
		{0, 4, 8},
		{0, 8},
	} {
		head, ok := sl.heads[level]
		if !ok {
			t.Fatalf("level %d has no head!", level)
		}
		var actual []int64
		for {
			got, err := head.value.Int64()
			if err != nil {
				t.Fatal(err)
			}
			actual = append(actual, got)
			if head.next[level] == nil {
				break
			}
			head = head.next[level]
		}
		if diff := cmp.Diff(items, actual); diff != "" {
			t.Errorf("level %d: incorrect items: %s", level, diff)
		}
	}
}

func BenchmarkSkipListAppend(b *testing.B) {
	// this benchmark proves that we're O(1) for appending an entry
	for bound := 0; bound < 21; bound += 4 {
		b.Run(fmt.Sprintf("after %d items", 1<<bound), func(b *testing.B) {
			b.StopTimer()
			sl := &skipList{
				RWMutex: &sync.RWMutex{},
				heads:   map[int]*skipListNode{},
				tails:   map[int]*skipListNode{},
				tail:    0,
			}
			for j := 0; j < 1<<bound; j++ {
				sl.append(newEvent(j))
			}
			inputs := make([]*event, b.N)
			for i := 0; i < b.N; i++ {
				inputs = append(inputs, newEvent(i))
			}
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				sl.append(inputs[i])
			}
		})
	}
}

func TestSkipListFindExact(t *testing.T) {
	sl := &skipList{
		RWMutex: &sync.RWMutex{},
		heads:   map[int]*skipListNode{},
		tails:   map[int]*skipListNode{},
		tail:    0,
	}
	for i := 0; i < 9; i++ {
		sl.append(newEvent(i))
	}

	for i := 0; i < 9; i++ {
		val := apd.New(int64(i), 0)
		node := sl.find(val)
		if node == nil {
			t.Fatalf("skip-list could not find %d", i)
		}
		if node.value.Cmp(val) != 0 {
			t.Fatalf("skip-list found %s, not %s", node.value.String(), val.String())
		}
	}
}

func TestSkipListFindInexact(t *testing.T) {
	sl := &skipList{
		RWMutex: &sync.RWMutex{},
		heads:   map[int]*skipListNode{},
		tails:   map[int]*skipListNode{},
		tail:    0,
	}
	for i := 0; i <= 20; i += 2 {
		sl.append(newEvent(i))
	}

	for i := 1; i < 20; i += 2 {
		val := apd.New(int64(i), 0)
		node := sl.find(val)
		if node == nil {
			t.Fatalf("skip-list could not find %d", i)
		}
		if node.value.Cmp(val) <= 0 {
			t.Fatalf("skip-list found %s, should be strictly larger than %s", node.value.String(), val.String())
		}
	}
}

func TestSkipListFindOutOfBounds(t *testing.T) {
	sl := &skipList{
		RWMutex: &sync.RWMutex{},
		heads:   map[int]*skipListNode{},
		tails:   map[int]*skipListNode{},
		tail:    0,
	}
	for i := 10; i <= 20; i += 2 {
		sl.append(newEvent(i))
	}

	for i := 0; i < 10; i++ {
		val := apd.New(int64(i), 0)
		node := sl.find(val)
		if node == nil {
			t.Fatalf("skip-list could not find %d", i)
		}
		if node.value.Cmp(val) <= 0 {
			t.Fatalf("skip-list found %s, should be strictly larger than %s", node.value.String(), val.String())
		}
	}
	for i := 21; i < 31; i++ {
		val := apd.New(int64(i), 0)
		node := sl.find(val)
		if node != nil {
			t.Fatalf("skip-list found %s for %d, should be nothing", node.value.String(), i)
		}
	}
}

func BenchmarkSkipListFind(b *testing.B) {
	// this benchmark proves that we're O(log(n)) for finds
	// TODO: we're somehow superlogarithmic but sublinear here, how?
	for bound := 0; bound < 21; bound += 1 {
		b.Run(fmt.Sprintf("after %d items", 1<<bound), func(b *testing.B) {
			b.StopTimer()
			sl := &skipList{
				RWMutex: &sync.RWMutex{},
				heads:   map[int]*skipListNode{},
				tails:   map[int]*skipListNode{},
				tail:    0,
			}
			for j := 0; j < 1<<bound; j++ {
				sl.append(newEvent(j))
			}
			inputs := make([]*apd.Decimal, b.N)
			for i := 0; i < b.N; i++ {
				inputs[i] = apd.New(rand.Int63n(1<<bound), 0)
			}
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				sl.find(inputs[i])
			}
		})
	}
}

func TestSkipListResetHead(t *testing.T) {
	sl := &skipList{
		RWMutex: &sync.RWMutex{},
		heads:   map[int]*skipListNode{},
		tails:   map[int]*skipListNode{},
		tail:    0,
	}
	var nodes []*skipListNode
	for i := 0; i < 9; i++ {
		nodes = append(nodes, sl.append(newEvent(i)))
	}
	sl.resetHead(nodes[3])
	// we expect the following pointers:
	// 0: 3 -> 4 -> 5 -> 6 -> 7 -> 8 -> nil
	// 1: 3 -> 4 ------> 6 ------> 8 -> nil
	// 2: 3 -> 4 ----------------> 8 -> nil
	// 3: 3 ---------------------> 8 -> nil
	for level, items := range [][]int64{
		{3, 4, 5, 6, 7, 8},
		{3, 4, 6, 8},
		{3, 4, 8},
		{3, 8},
	} {
		head, ok := sl.heads[level]
		if !ok {
			t.Fatalf("level %d has no head!", level)
		}
		var actual []int64
		for {
			got, err := head.value.Int64()
			if err != nil {
				t.Fatal(err)
			}
			actual = append(actual, got)
			if head.next[level] == nil {
				break
			}
			head = head.next[level]
		}
		if diff := cmp.Diff(items, actual); diff != "" {
			t.Errorf("level %d: incorrect items: %s", level, diff)
		}
	}
}

type fakeQuerier struct {
	validate func(sql string, args ...interface{})
	rows     *fakeRows
	err      error
}

func (f *fakeQuerier) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if validate := f.validate; validate != nil {
		f.validate = nil // consume the func so we don't hotloop
		validate(sql, args...)
	}
	if err := f.err; err != nil {
		f.err = nil // consume the error so we don't hotloop
		return nil, err
	}
	return f.rows, nil
}

type fakeRows struct {
	next chan bool
	errs chan error
	data chan [][]byte
}

func (f *fakeRows) Close() {}

func (f *fakeRows) Err() error {
	return <-f.errs
}

func (f *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (f *fakeRows) FieldDescriptions() []pgproto3.FieldDescription {
	return nil
}

func (f *fakeRows) Next() bool {
	return <-f.next
}

func (f *fakeRows) Scan(dest ...interface{}) error {
	return errors.New("not implemented")
}

func (f *fakeRows) Values() ([]interface{}, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeRows) RawValues() [][]byte {
	return <-f.data
}

func (f *fakeRows) sendError(err error) {
	f.next <- true
	f.errs <- err
}

func (f *fakeRows) sendRow(data []byte) {
	f.next <- true
	f.errs <- nil
	f.data <- [][]byte{{}, {}, data}
}

func (f *fakeRows) close() {
	f.next <- false
}

func serialize(t *testing.T, event *changefeedEvent) []byte {
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("could not serialize event: %v", err)
	}
	return raw
}

func TestChangefeedCacheIntegration(t *testing.T) {
	rows := &fakeRows{
		next: make(chan bool),
		errs: make(chan error),
		data: make(chan [][]byte),
	}
	q := &fakeQuerier{
		rows: rows,
	}
	cache := newChangefeedCache(context.Background(), q)

	t.Log("send some events, out of order")
	for _, i := range []int{1, 2, 6, 4, 5, 3, 9, 8, 7} {
		rows.sendRow(serialize(t, &changefeedEvent{MVCCTimestamp: apd.New(int64(i), 0)}))
	}

	t.Log("wait until the cache has seen everything we sent it")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cache.waitForSync(ctx, apd.New(int64(7), 0))

	t.Log("expect the linked list to hold the order we sent")
	{
		var items []string
		head := cache.buffer.head
		for head != nil {
			items = append(items, head.data.rawRev.String())
			head = head.next
		}
		if diff := cmp.Diff(items, []string{"1", "2", "6", "4", "5", "3", "9", "8", "7"}); diff != "" {
			t.Fatalf("linked list was not in the order that events were received: %v", diff)
		}
	}

	t.Log("however, the buffer should be ordered")
	{
		var items []string
		for i := range cache.buffer.items {
			items = append(items, cache.buffer.items[i].data.rawRev.String())
		}
		if diff := cmp.Diff(items, []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}); diff != "" {
			t.Fatalf("buffer was not in sorted order: %v", diff)
		}
	}

	t.Log("send some more events, out of order")
	for _, i := range []int{11, 12, 16, 14, 15, 13} {
		rows.sendRow(serialize(t, &changefeedEvent{MVCCTimestamp: apd.New(int64(i), 0)}))
	}

	t.Log("mark the first batch of events resolved")
	rows.sendRow(serialize(t, &changefeedEvent{Resolved: apd.New(int64(10), 0)}))

	t.Log("send a new event after the resolution marker")
	rows.sendRow(serialize(t, &changefeedEvent{MVCCTimestamp: apd.New(int64(17), 0)}))

	t.Log("wait until the cache has seen everything we sent it")
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	cache.waitForSync(ctx, apd.New(int64(17), 0))

	t.Log("expect the linked list to hold only the unresolved events in the order we sent")
	{
		var items []string
		head := cache.buffer.head
		for head != nil {
			items = append(items, head.data.rawRev.String())
			head = head.next
		}
		if diff := cmp.Diff(items, []string{"11", "12", "16", "14", "15", "13", "10", "17"}); diff != "" {
			t.Fatalf("linked list was not in the order that events were received: %v", diff)
		}
	}

	t.Log("however, the buffer should be ordered")
	{
		var items []string
		for i := range cache.buffer.items {
			items = append(items, cache.buffer.items[i].data.rawRev.String())
		}
		if diff := cmp.Diff(items, []string{"11", "12", "13", "14", "15", "16", "17"}); diff != "" {
			t.Fatalf("buffer was not in sorted order: %v", diff)
		}
	}

	t.Log("as should the history, which contains up to and including the resolution marker")
	{
		var items []string
		head := cache.history.heads[0]
		for head != nil {
			items = append(items, head.data.rawRev.String())
			head = head.next[0]
		}
		if diff := cmp.Diff(items, []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}); diff != "" {
			t.Fatalf("history was not in sorted order: %v", diff)
		}
	}

	type metadata struct {
		events chan *event
		sink   []string
		ctx    context.Context
		cancel func()
	}
	watchers := map[int]*metadata{}
	for _, start := range []int{5, 12} {
		subCtx, subCancel := context.WithTimeout(context.Background(), 10*time.Second)
		meta := &metadata{
			events: make(chan *event),
			ctx:    subCtx,
			cancel: subCancel,
		}
		watchers[start] = meta
	}

	t.Log("start some watchers, one from the history and one from the buffer")
	for start, meta := range watchers {
		go cache.subscribeAfter(meta.ctx, apd.New(int64(start), 0), meta.events)
	}

	t.Log("drain the watch channels and stop watching once we saw everything")
	for start := range watchers {
		meta := watchers[start]
		func() {
			for {
				select {
				case <-meta.ctx.Done():
					return
				case evt := <-meta.events:
					meta.sink = append(meta.sink, evt.rawRev.String())
					if evt.rawRev.Cmp(apd.New(int64(17), 0)) == 0 {
						meta.cancel()
						return
					}
				}
			}
		}()
	}

	expected := map[int][]string{
		5:  {"5", "6", "7", "8", "9", "10", "11", "12", "16", "14", "15", "13", "10", "17"}, // TODO: so we're double-delivering the resolved event
		12: {"12", "16", "14", "15", "13", "17"},
	}
	for start, events := range expected {
		if diff := cmp.Diff(events, watchers[start].sink); diff != "" {
			t.Errorf("watcher from %d got incorrect events: %v", start, diff)
		}
	}

	rows.close()
}

func TestChangefeedCacheCancellation(t *testing.T) {
	rows := &fakeRows{
		next: make(chan bool),
		errs: make(chan error),
		data: make(chan [][]byte),
	}
	q := &fakeQuerier{
		rows: rows,
	}
	t.Log("start up a cache, feed it no events")
	ctx, cancel := context.WithCancel(context.Background())
	cache := newChangefeedCache(ctx, q)

	t.Log("start a client waiting for an event it will never see, using a long client timeout")
	done := make(chan struct{})
	go func() {
		ctx, _ := context.WithTimeout(context.Background(), 100*time.Second)
		cache.waitForSync(ctx, apd.New(int64(10), 0))
		done <- struct{}{}
	}()

	t.Log("cancel the cache's context")
	cancel()

	select {
	case <-time.After(10 * time.Second):
		t.Fatal("waiter on cache sync did not exit on cancellation")
	case <-done:
	}
}

func TestChangefeedCacheClientCancellation(t *testing.T) {
	rows := &fakeRows{
		next: make(chan bool),
		errs: make(chan error),
		data: make(chan [][]byte),
	}
	q := &fakeQuerier{
		rows: rows,
	}
	t.Log("start up a cache, feed it no events")
	cache := newChangefeedCache(context.Background(), q)

	t.Log("start a client waiting for an event it will never see")
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		cache.waitForSync(ctx, apd.New(int64(10), 0))
		done <- struct{}{}
	}()

	t.Log("cancel the client's context")
	cancel()

	select {
	case <-time.After(10 * time.Second):
		t.Fatal("waiter on cache sync did not exit on client context cancellation")
	case <-done:
	}
}

func TestChangefeedCacheEventDeduplication(t *testing.T) {
	rows := &fakeRows{
		next: make(chan bool),
		errs: make(chan error),
		data: make(chan [][]byte),
	}
	q := &fakeQuerier{
		rows: rows,
	}
	cache := newChangefeedCache(context.Background(), q)

	t.Log("send some events, some with duplicates")
	for _, i := range []int{1, 2, 2, 6, 4, 6, 5} {
		rows.sendRow(serialize(t, &changefeedEvent{MVCCTimestamp: apd.New(int64(i), 0)}))
	}

	t.Log("wait until the cache has seen everything we sent it")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cache.waitForSync(ctx, apd.New(int64(5), 0))

	t.Log("start a watcher")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	events := make(chan *event)
	go cache.subscribeAfter(ctx, apd.New(int64(1), 0), events)

	t.Log("drain the watch channels and stop watching once we saw everything")
	var sink []string
	func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-events:
				sink = append(sink, evt.rawRev.String())
				if evt.rawRev.Cmp(apd.New(int64(5), 0)) == 0 {
					cancel()
					return
				}
			}
		}
	}()

	expected := []string{"1", "2", "6", "4", "5"}
	if diff := cmp.Diff(expected, sink); diff != "" {
		t.Errorf("watcher got incorrect events: %v", diff)
	}

	rows.close()
}

func TestChangefeedCacheAdjacentResolvedEvents(t *testing.T) {
	rows := &fakeRows{
		next: make(chan bool),
		errs: make(chan error),
		data: make(chan [][]byte),
	}
	q := &fakeQuerier{
		rows: rows,
	}
	cache := newChangefeedCache(context.Background(), q)

	t.Log("send enough resolved events to trigger garbage collection")
	var i int
	var expected []string
	for i = 1; i <= len(cache.resolutionMarkers)+1; i++ {
		rows.sendRow(serialize(t, &changefeedEvent{Resolved: apd.New(int64(i), 0)}))
		expected = append(expected, strconv.Itoa(i))
	}

	t.Log("wait until the cache has seen everything we sent it")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cache.waitForSync(ctx, apd.New(int64(i-1), 0))

	t.Log("start a watcher")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	events := make(chan *event)
	go cache.subscribeAfter(ctx, apd.New(int64(1), 0), events)

	t.Log("drain the watch channels and stop watching once we saw everything")
	var sink []string
	func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-events:
				sink = append(sink, evt.rawRev.String())
				if evt.rawRev.Cmp(apd.New(int64(i-1), 0)) == 0 {
					cancel()
					return
				}
			}
		}
	}()

	if diff := cmp.Diff(expected, sink); diff != "" {
		t.Errorf("watcher got incorrect events: %v", diff)
	}

	rows.close()
}

func TestChangefeedCacheGracefulReconnection(t *testing.T) {
	rows := &fakeRows{
		next: make(chan bool),
		errs: make(chan error),
		data: make(chan [][]byte),
	}
	q := &fakeQuerier{
		rows: rows,
	}
	t.Log("start up a cache")
	cache := newChangefeedCache(context.Background(), q)

	t.Log("send some events")
	for _, i := range []int{1, 2, 6, 4, 5, 3, 9, 8, 7} {
		rows.sendRow(serialize(t, &changefeedEvent{MVCCTimestamp: apd.New(int64(i), 0)}))
	}
	rows.sendRow(serialize(t, &changefeedEvent{Resolved: apd.New(int64(10), 0)}))

	t.Log("wait until the cache has seen everything we sent it")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cache.waitForSync(ctx, apd.New(int64(7), 0))

	t.Log("start a watcher")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	events := make(chan *event)
	go cache.subscribeAfter(ctx, apd.New(int64(1), 0), events)

	t.Log("set up a validator to ensure we see the correct reconnection")
	q.validate = func(sql string, args ...interface{}) {
		cursor := "cursor='10'"
		if !strings.Contains(sql, cursor) {
			t.Fatalf("expected %q in the reconnection command, got %q", cursor, sql)
		}
	}

	t.Log("send an error on the changefeed stream")
	rows.sendError(errors.New("oops"))

	t.Log("send some more new events")
	for _, i := range []int{14, 12, 11} {
		rows.sendRow(serialize(t, &changefeedEvent{MVCCTimestamp: apd.New(int64(i), 0)}))
	}

	t.Log("wait until the cache has seen everything we sent it")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	cache.waitForSync(ctx, apd.New(int64(11), 0))

	t.Log("drain the watch channels and stop watching once we saw everything")
	var sink []string
	func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-events:
				sink = append(sink, evt.rawRev.String())
				if evt.rawRev.Cmp(apd.New(int64(11), 0)) == 0 {
					cancel()
					return
				}
			}
		}
	}()

	expected := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "14", "12", "11"}
	if diff := cmp.Diff(expected, sink); diff != "" {
		t.Errorf("watcher got incorrect events: %v", diff)
	}

	rows.close()
}

func TestChangefeedCacheFailedReconnection(t *testing.T) {
	rows := &fakeRows{
		next: make(chan bool),
		errs: make(chan error),
		data: make(chan [][]byte),
	}
	q := &fakeQuerier{
		rows: rows,
	}
	t.Log("start up a cache")
	cache := newChangefeedCache(context.Background(), q)

	t.Log("send some events")
	for _, i := range []int{1, 2, 6, 4, 5, 3, 9, 8, 7} {
		rows.sendRow(serialize(t, &changefeedEvent{MVCCTimestamp: apd.New(int64(i), 0)}))
	}
	rows.sendRow(serialize(t, &changefeedEvent{Resolved: apd.New(int64(10), 0)}))

	t.Log("wait until the cache has seen everything we sent it")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cache.waitForSync(ctx, apd.New(int64(10), 0))

	t.Log("start a watcher")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	events := make(chan *event)
	go cache.subscribeAfter(ctx, apd.New(int64(1), 0), events)

	t.Log("drain the watch channel for the events we know we should see")
	var sink []string
	func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-events:
				sink = append(sink, evt.rawRev.String())
				if evt.rawRev.Cmp(apd.New(int64(10), 0)) == 0 {
					return
				}
			}
		}
	}()

	t.Log("set up a validator to ensure we see the correct reconnection")
	q.validate = func(sql string, args ...interface{}) {
		cursor := "cursor='10'"
		if !strings.Contains(sql, cursor) {
			t.Fatalf("expected %q in the reconnection command, got %q", cursor, sql)
		}
	}

	t.Log("set up the querier to fail the reconnection attempt")
	q.err = &pgconn.PgError{
		Severity: "FATAL",
		Code:     "XXUUU",
		Message:  "batch timestamp 10 must be after replica GC threshold 11",
	}

	t.Log("send an error on the changefeed stream")
	rows.sendError(errors.New("oops"))

	t.Log("drain the watch channel and stop watching once we saw everything")
	func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-events:
				if evt.eType == eventTypeCompacted {
					cancel()
					return
				}
				sink = append(sink, evt.rawRev.String())
			}
		}
	}()

	expected := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	if diff := cmp.Diff(expected, sink); diff != "" {
		t.Errorf("watcher got incorrect events: %v", diff)
	}

	t.Log("send some events, since the cache should have restarted behind the scenes")
	for _, i := range []int{1, 2, 6, 4, 5, 3, 9, 8, 7} {
		rows.sendRow(serialize(t, &changefeedEvent{MVCCTimestamp: apd.New(int64(i), 0)}))
	}
	rows.sendRow(serialize(t, &changefeedEvent{Resolved: apd.New(int64(10), 0)}))

	t.Log("wait until the cache has seen everything we sent it")
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	cache.waitForSync(ctx, apd.New(int64(10), 0))

	rows.close()
}
