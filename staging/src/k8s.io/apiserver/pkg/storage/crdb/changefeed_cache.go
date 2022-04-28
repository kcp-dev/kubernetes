package crdb

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/apd"
	"github.com/jackc/pgx/v4"

	"k8s.io/apiserver/pkg/storage/generic"
	"k8s.io/klog/v2"
)

// we open one changefeed for the entire `k8s` table, per API server
// we buffer incoming events and as new ones come in, place them in correct append order in the buffer
// when we get a resolved timestamp, we flush the buffer up to that HLC to some larger data store
// in the larger data store, we can collapse consequent resolved timestamps and just keep the newest?
// the larger data store should be a ttl-cache where we retain last 5min or whatever configured compaction interval

// clients to this will only reach us after their initial list, so they just come in with some HLC they need
// maybe need a two-stage thing but ultimately determine if their incoming HLC request is in the buffer or the store
// stream the events we have to them and add them to the list of consumers for new events
// we likely do not want to store per-client info on where they are in consuming events ... need to synchronize around
// the transition between backfill from our cache and being on the recieving end of the incoming stuff

// incidentally this sort of approach with sorting will make it easy to find where we need to start serving,
// and will give total order of events but only for cached items

// what if we hold linked-lists - one for the history, and one for the buffer, but in order of events
// then, when we move from buffer to history we rewrite the pointers so history is always in order
// consumers get a goroutine that traverses history until they get to the end of it *as it was when initiated*
// then continue on to the buffer, in order of events, at the moment they saw it
// this reduces any blocking issues since each thread continues at its own consumption pace
// however, it will rely heavily on the garbage collector and slow consumers will increase memory footprint
// how to sync when they get to the head of the linked list and need to wait for the next event to come in?

var lock = &sync.Mutex{}
var cache *changefeedCache

func initialize(ctx context.Context, client querier) *changefeedCache {
	// TODO: can this be more intelligently a singleton?
	lock.Lock()
	if cache == nil {
		now := time.Now()
		cache = newChangefeedCache(ctx, now, client)
		cache.waitForSync(ctx, apd.New(now.UnixNano(), 0))
	}
	c := cache
	lock.Unlock()
	return c
}

type watchEvent struct {
	*generic.WatchResponse
	timestamp *apd.Decimal
}

type changefeedEvent struct {
	MVCCTimestamp *apd.Decimal `json:"mvcc_timestamp,omitempty"`
	Resolved      *apd.Decimal `json:"resolved,omitempty"`

	Before *row `json:"before,omitempty"`
	After  *row `json:"after,omitempty"`
}

type row struct {
	Key     string `json:"key,omitempty"`
	Value   string `json:"value,omitempty"`
	Cluster string `json:"cluster,omitempty"`
}

type changefeedCache struct {
	ctx context.Context

	bufferLock *sync.RWMutex
	// buffer holds incoming events we have not yet flushed to history
	buffer *linkedList
	// highWaterMark is the most recent timestamp we've consumed
	highWaterMark *apd.Decimal

	// history holds sorted history of events for long-term storage
	history *skipList

	// markerLock guards the store of resolution markers
	// ALWAYS hold bufferLock when acquiring this to avoid deadlock
	markerLock *sync.RWMutex
	// markerIndex is the last resolution marker we've written
	markerIndex int
	// resolutionMarkers holds records of all resolved intervals we've seen
	resolutionMarkers []*skipListNode

	// incoming gets a broadcast when we process an event
	incoming *sync.Cond
	// clients lets us wait for client connections to die down
	clients *sync.WaitGroup
	// compacted lets us signal to all clients to hang up
	compacted chan struct{}
}

type querier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

const (
	resolveInterval = 100 * time.Millisecond // TODO: plumb this in
	ttlInterval     = 5 * time.Minute        // TODO: plumb this in
)

func newChangefeedCache(ctx context.Context, startAt time.Time, client querier) *changefeedCache {
	cache := changefeedCache{
		ctx:               ctx,
		bufferLock:        &sync.RWMutex{},
		buffer:            newLinkedList(),
		history:           newSkipList(),
		markerLock:        &sync.RWMutex{},
		resolutionMarkers: make([]*skipListNode, ttlInterval/resolveInterval),
		incoming:          sync.NewCond(&sync.Mutex{}),
		clients:           &sync.WaitGroup{},
		compacted:         make(chan struct{}),
	}

	wg := &sync.WaitGroup{}
	events := make(chan *watchEvent, 100)
	wg.Add(1)
	go func(ctx context.Context, client querier, events chan<- *watchEvent, cache *changefeedCache) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			cache.bufferLock.RLock()
			highWaterMark := cache.highWaterMark
			cache.bufferLock.RUnlock()
			if highWaterMark == nil {
				// this is the first time the cache is starting, so we should fill in the needed info
				var created apd.Decimal
				if err := client.QueryRow(ctx, `SELECT mod_time_logical FROM crdb_internal.tables WHERE name='k8s';`).Scan(&created); err != nil {
					klog.Errorf("could not determine table creation: %v", err)
					continue
				}
				// TODO: This is racy since there's time between us calculating this interval
				// (and fetching the above creation time) and the moment we start the changefeed,
				// in the intervening time the GC TTL may have passed. In fact, due to how we are
				// calculating this interval, it is likely this may occur. Can we do better?
				// For now, we can calculate a slight offset here...
				recentInterval := apd.New(startAt.Add(-ttlInterval+5*time.Second).UnixNano(), 0)
				if recentInterval.Cmp(&created) > 0 {
					// the table was created more than the TTL interval ago, so we should just start watching then
					highWaterMark = recentInterval
				} else {
					// one full TTL interval ago was before the table was created, so the earliest we can watch is the creation time
					highWaterMark = &created
				}
			}
			if consumeChangefeed(ctx, client, highWaterMark, events) {
				// we need to invalidate the cache and restart, but first let all clients know to go away
				close(cache.compacted)
				cache.incoming.Broadcast()
				cache.clients.Wait()

				// now, invalidate all of the previous data and restart
				cache.bufferLock.Lock()
				cache.markerLock.Lock()
				cache.buffer = newLinkedList()
				cache.history = newSkipList()
				cache.resolutionMarkers = make([]*skipListNode, ttlInterval/resolveInterval)
				cache.highWaterMark = nil
				cache.compacted = make(chan struct{})
				cache.markerLock.Unlock()
				cache.bufferLock.Unlock()
			}
		}
	}(ctx, client, events, &cache)

	wg.Add(1)
	go func(ctx context.Context, cache *changefeedCache, events <-chan *watchEvent) {
		defer wg.Done()
		for {
			select {
			case evt := <-events:
				cache.process(evt)
			case <-ctx.Done():
				return
			}
		}
	}(ctx, &cache, events)

	go func(ctx context.Context, cond *sync.Cond) {
		<-ctx.Done()
		// waiters on sync.Cond only wake up when the condition is signalled, so we need to
		// broadcast after the context is finished to wake up any waiters, so they can notice
		// the context cancellation and exit
		cond.Broadcast()
		wg.Wait()
		close(events)
	}(ctx, cache.incoming)

	return &cache
}

// consumeChangefeed creates a changefeed and consumes all events from it, returning if something goes wrong.
// The `terminal` return value indicates whether the cache should be invalidated or if the last known timestamp
// can be used to start a new changefeed.
func consumeChangefeed(ctx context.Context, client querier, timestamp *apd.Decimal, events chan<- *watchEvent) (terminal bool) {
	options := []string{"diff", "mvcc_timestamp", fmt.Sprintf("resolved='%s'", resolveInterval.String())}
	if timestamp != nil {
		options = append(options, fmt.Sprintf("no_initial_scan,cursor='%s'", timestamp.String()))
	}
	query := fmt.Sprintf(`EXPERIMENTAL CHANGEFEED FOR k8s WITH %s;`, strings.Join(options, ","))
	rows, err := client.Query(ctx, query)
	if err != nil {
		klog.Errorf("could not initialize changefeed: %v", err)
		// if the timestamp has been compacted, we need to refresh the cache entirely
		return isGarbageCollectionError(err)
	}
	defer func() {
		rows.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		if !rows.Next() {
			break
		}
		if err := rows.Err(); err != nil {
			klog.Errorf("failed to read changefeed row: %v", err)
			return false
		}

		values := rows.RawValues()
		if len(values) != 3 {
			klog.Errorf("expected 3 values in changefeed row, got %d", len(values))
			return false
		}

		// values unpack into (tableName, primaryKey, rowData)
		// tableName = name
		// primaryKey = ["array", "of", "values"]
		// rowData = {...}
		data := values[2]

		var evt changefeedEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			klog.Errorf("failed to deserialize changefeed row: %v", err)
			return false
		}

		parsed, err := parseEvent(&evt)
		if err != nil {
			klog.Errorf("failed to parse changefeed row: %v", err)
			return false
		}

		select {
		case events <- parsed:
		case <-ctx.Done():
			return false
		}

	}
	return false
}

func parseEvent(e *changefeedEvent) (*watchEvent, error) {
	var timestamp *apd.Decimal
	if e.Resolved != nil {
		timestamp = e.Resolved
	} else {
		timestamp = e.MVCCTimestamp
	}
	resourceVersion, err := toResourceVersion(timestamp)
	if err != nil {
		return nil, err
	}
	resp := &watchEvent{
		WatchResponse: &generic.WatchResponse{
			Header:         &generic.ResponseHeader{Revision: resourceVersion},
			ProgressNotify: e.Resolved != nil,
		},
		timestamp: timestamp,
	}
	if e.Resolved == nil {
		var key string
		switch {
		case e.Before != nil:
			key = e.Before.Key
		case e.After != nil:
			key = e.After.Key
		}

		// etcd always sends the non-nil keys in `kv` and `prevkv`, so adapt the CRDB response
		evt := &generic.WatchEvent{
			Kv:     &generic.KeyValue{Key: []byte(key), ModRevision: resourceVersion},
			PrevKv: &generic.KeyValue{Key: []byte(key), ModRevision: resourceVersion},
		}
		for source, sink := range map[*row]*generic.KeyValue{
			e.Before: evt.PrevKv,
			e.After:  evt.Kv,
		} {
			if source != nil {
				value, err := hex.DecodeString(strings.TrimPrefix(source.Value, "\\x"))
				if err != nil {
					return nil, fmt.Errorf("failed to decode timestamp: %w", err)
				}
				sink.Value = value
			}
		}
		switch {
		case e.After == nil:
			evt.Type = generic.EventTypeDelete
		case e.Before == nil:
			evt.Type = generic.EventTypeCreate
		default:
			evt.Type = generic.EventTypeModify
		}
		resp.Events = []*generic.WatchEvent{evt}
	}
	return resp, nil
}

func (c *changefeedCache) process(evt *watchEvent) {
	c.bufferLock.Lock()
	// find the index where we need to append this into the buffer
	i := sort.Search(len(c.buffer.items), func(i int) bool { return c.buffer.items[i].timestamp.Cmp(evt.timestamp) >= 0 })
	if i < len(c.buffer.items) && c.buffer.items[i].timestamp.Cmp(evt.timestamp) == 0 {
		// this is a duplicate event, ignore it
		// we get duplicates in one of two scenarios:
		// - first, rarely from CRDB itself
		// - second, when we've seen an error on the changefeed stream and needed to reconnect,
		//   as we can only reconnect from the last resolved timestamp in order to not lose events;
		//   therefore we will replay the buffer, perhaps in some other order
		c.bufferLock.Unlock()
		return
	}

	// add to the tail of the linked list
	evtNode := &linkedListNode{timestamp: evt.timestamp, data: evt.WatchResponse}
	if c.buffer.head == nil {
		c.buffer.head = evtNode
	}
	tail := c.buffer.tail
	if tail != nil {
		tail.next = evtNode
	}
	c.buffer.tail = evtNode

	// append into buffer, sorted
	if i == len(c.buffer.items) {
		c.buffer.items = append(c.buffer.items, evtNode)
	} else {
		c.buffer.items = append(c.buffer.items[:i], append([]*linkedListNode{evtNode}, c.buffer.items[i:]...)...)
	}

	if evt.ProgressNotify {
		c.highWaterMark = evt.timestamp
		// move events up to the resolved timestamp into storage
		var marker *skipListNode
		for _, buffered := range c.buffer.items[:i+1] {
			if node := c.history.append(buffered.timestamp, buffered.data); buffered.timestamp.Cmp(evt.timestamp) == 0 {
				marker = node
			}
		}
		// remove those events from the buffer
		if i+1 == len(c.buffer.items) {
			// we're flushing everything
			c.buffer.head = nil
			c.buffer.items = []*linkedListNode{}
		} else {
			c.buffer.head = c.buffer.items[i+1]
			c.buffer.items = c.buffer.items[i+1:]
		}

		// update our record markers and GC old data
		c.markerLock.Lock()
		if c.markerIndex == len(c.resolutionMarkers)-1 {
			// pop the oldest data out of the cache
			c.history.resetHead(c.resolutionMarkers[0])
			c.resolutionMarkers = c.resolutionMarkers[1:]
		}
		c.resolutionMarkers[c.markerIndex] = marker
		// increment until we latch onto the last index
		if c.markerIndex < len(c.resolutionMarkers)-2 {
			c.markerIndex++
		}
		c.markerLock.Unlock()
	}

	// let consumers know that there's something to read
	c.incoming.Broadcast()
	c.bufferLock.Unlock()
}

// waitForSync waits until the cache has seen something at least as old as timestamp
func (c *changefeedCache) waitForSync(ctx context.Context, timestamp *apd.Decimal) bool {
	c.incoming.L.Lock()
	defer c.incoming.L.Unlock()
	for {
		select {
		case <-c.ctx.Done():
			return false // cache shutting down
		case <-c.compacted:
			return true // cache needs to revalidate
		case <-ctx.Done():
			return false // user cancelled their watch
		default:
		}
		c.bufferLock.RLock()
		// have we resolved some timestamp after the event?
		eventResolved := c.highWaterMark != nil && c.highWaterMark.Cmp(timestamp) >= 0
		var eventBuffered bool
		if !eventResolved {
			// is the event in our buffer?
			idx := sort.Search(len(c.buffer.items), func(i int) bool { return c.buffer.items[i].timestamp.Cmp(timestamp) >= 0 })
			eventBuffered = idx < len(c.buffer.items) && c.buffer.items[idx].timestamp.Cmp(timestamp) == 0
		}
		c.bufferLock.RUnlock()
		if eventResolved || eventBuffered {
			break
		}
		c.incoming.Wait()
	}
	return false
}

func (c *changefeedCache) subscribeAfter(ctx context.Context, timestamp *apd.Decimal, events chan<- *generic.WatchResponse) {
	c.clients.Add(1)
	defer c.clients.Done()
	compacted := c.waitForSync(ctx, timestamp)

	c.bufferLock.RLock()
	if compacted || (c.history.heads[0] != nil && timestamp.Cmp(c.history.heads[0].timestamp) < 0) {
		events <- &generic.WatchResponse{Error: generic.ErrCompacted}
		return
	}

	// search the cache to figure out from where we start sending events
	var historyStart *skipListNode
	var bufferStart *linkedListNode
	if len(c.buffer.items) == 0 || timestamp.Cmp(c.buffer.items[0].timestamp) < 0 {
		// requested timestamp is in history somewhere, since buffer is empty and we waited for the event to have been seen
		historyStart = c.history.find(timestamp)
		bufferStart = c.buffer.head
	} else {
		bufferIndex := sort.Search(len(c.buffer.items), func(i int) bool { return c.buffer.items[i].timestamp.Cmp(timestamp) >= 0 })
		if bufferIndex < len(c.buffer.items) { // otherwise, we have not yet seen it?
			bufferStart = c.buffer.items[bufferIndex]
		}
	}
	c.bufferLock.RUnlock()

	// our data is read-only, and the `.next` pointers in our cache have a single
	// writer that converges, so we can traverse these lists without locking

	// start by draining the history we need
	for historyStart != nil {
		events <- historyStart.data
		historyStart = historyStart.next[0]
	}
	// then, follow along with the incoming events
	if bufferStart == nil {
		// if two resolved timestamps are received in order, we will have an empty buffer
		// and will need to wait until we see *something* in order to serve it here ...
		c.incoming.L.Lock()
		for {
			select {
			case <-c.ctx.Done():
				c.incoming.L.Unlock()
				return // cache shutting down
			case <-c.compacted:
				c.incoming.L.Unlock()
				events <- &generic.WatchResponse{Error: generic.ErrCompacted}
				return
			case <-ctx.Done():
				c.incoming.L.Unlock()
				return // user cancelled their watch
			default:
			}
			if c.buffer.head != nil {
				break
			}
			c.incoming.Wait()
		}
		bufferStart = c.buffer.head
		c.incoming.L.Unlock()
	}
	events <- bufferStart.data
	var timestampResolved bool
	for {
		select {
		case <-c.ctx.Done():
			return // cache shutting down
		case <-c.compacted:
			events <- &generic.WatchResponse{Error: generic.ErrCompacted}
			return
		case <-ctx.Done():
			return // user cancelled their watch
		default:
		}
		c.incoming.L.Lock()
		for bufferStart.next == nil {
			c.incoming.Wait()
		}
		c.incoming.L.Unlock()

		bufferStart = bufferStart.next
		// we are streaming things out of order, and if we've been requested to start somewhere
		// in the middle of the buffer, we need to filter out-of-order nodes out to be correct
		if timestampResolved || timestamp.Cmp(bufferStart.timestamp) <= 0 {
			timestampResolved = bufferStart.data.ProgressNotify // stop checking once we know everything else will be true
			events <- bufferStart.data
		}
		// TODO: re-read k8s watch cache to see how they implement size bounds
		// TODO: is there a test for watch cache memory consumption?
	}
}

func newLinkedList() *linkedList {
	return &linkedList{}
}

type linkedList struct {
	head *linkedListNode
	tail *linkedListNode

	// items holds all of the nodes of the list in order to facilitate sorting
	items []*linkedListNode
}

type linkedListNode struct {
	next      *linkedListNode
	timestamp *apd.Decimal
	data      *generic.WatchResponse
}

// TSCache paper: http://www.vldb.org/pvldb/vol14/p3253-liu.pdf
// Probablistic SkipList paper: https://15721.courses.cs.cmu.edu/spring2018/papers/08-oltpindexes1/pugh-skiplists-cacm1990.pdf

func newSkipList() *skipList {
	return &skipList{
		RWMutex: &sync.RWMutex{},
		heads:   map[int]*skipListNode{},
		tails:   map[int]*skipListNode{},
	}
}

// skipList is a deterministic (!) implementation of a skip-list, optimized for holding time-series data.
// Based on the domain, we know we have read-only data that is inserted in order. We furthermore know that
// there is one and only one access pattern - finding the start of a range and iterating forward forever.
// We can therefore build a skip-list implementation for which insertion and prefix deletion are O(1)
// operations and find is O(log(n)). This structure holds 2n pointers.
type skipList struct {
	*sync.RWMutex
	largestLevel int
	heads        map[int]*skipListNode
	tails        map[int]*skipListNode
	tail         int
}

type skipListNode struct {
	next      map[int]*skipListNode // we know this will be small but sparse, don't want to allocate full array for every node
	timestamp *apd.Decimal
	data      *generic.WatchResponse
}

// append adds a new node for the item at the tail of the skip-list, while also filling in any necessary
// skip pointers for previous entries
func (l *skipList) append(timestamp *apd.Decimal, data *generic.WatchResponse) *skipListNode {
	node := skipListNode{
		next:      map[int]*skipListNode{},
		timestamp: timestamp,
		data:      data,
	}
	l.Lock()
	if l.tail == 1<<(l.largestLevel) {
		l.largestLevel++
	}
	for level := 0; level <= l.largestLevel; level++ {
		// if this a level we've not touched before, initialize the head&tail
		if l.heads[level] == nil {
			l.heads[level] = l.heads[0]
		}
		if l.tails[level] == nil {
			l.tails[level] = l.heads[0]
		}

		// for every level we match, update the tail to point to us and make us the new tail
		factor := 1 << level
		if l.tail%(factor) == 0 {
			tail := l.tails[level]
			if tail != nil {
				tail.next[level] = &node
			}
			l.tails[level] = &node

			if l.heads[level] == nil {
				l.heads[level] = &node
			}
		}
	}
	l.tail++
	l.Unlock()
	return &node
}

// find will return the node with the smallest value that is no smaller than the given value.
// Generally, the algorithm goes like this: starting with the largest level, iterate until
// you find a node with a value that's larger than the given value. Then, go to the next smallest
// level, and repeat. This is effectively a binary search from one end.
func (l *skipList) find(value *apd.Decimal) *skipListNode {
	l.RLock()
	defer l.RUnlock()
	level := l.largestLevel
	head := l.heads[level]
	for {
		next := head.next[level]
		if next != nil && next.timestamp.Cmp(value) <= 0 {
			// we can keep searching at this coarse-grained level
			head = next
			continue
		}
		if level == 0 {
			// no finer-grained level exists, we are either at the node or one below it
			if head.timestamp.Cmp(value) == 0 {
				return head
			}
			return next
		}
		// we need to head down a level for a finer-grained search
		level--
	}
}

// resetHead accepts a node that has already been inserted into the skip-list and truncates
// all nodes that come before it from the list
func (l *skipList) resetHead(newHead *skipListNode) {
	l.Lock()
	// wire our new head up to the next links it needs
	for level := 0; level <= l.largestLevel; level++ {
		head := l.heads[level]
		l.heads[level] = newHead
		if _, present := newHead.next[level]; present {
			// we already have a link at this level
			continue
		}

		for {
			next := head.next[level]
			if next != nil && next.timestamp.Cmp(newHead.timestamp) < 0 {
				// we can keep searching for a node that's larger than the new head
				head = next
				continue
			}
			// we've found the node at this level that is larger than our new head, wire it up
			newHead.next[level] = next
			break
		}
		if newHead.next[level] == nil {
			// no larger node exists at this level, so we're also the tail
			l.tails[level] = head
		}
	}
	l.Unlock()
}
