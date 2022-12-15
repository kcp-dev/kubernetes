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

package generic

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kcp-dev/logicalcluster/v2"
	"k8s.io/apimachinery/pkg/api/apitesting"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/apis/example"
	examplev1 "k8s.io/apiserver/pkg/apis/example/v1"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	utilflowcontrol "k8s.io/apiserver/pkg/util/flowcontrol"
)

func RunTestWatch(t *testing.T, client InternalTestClient) {
	testWatch(t, false, client)
	testWatch(t, true, client)
}

// It tests that
// - first occurrence of objects should notify Add event
// - update should trigger Modified event
// - update that gets filtered should trigger Deleted event
func testWatch(t *testing.T, recursive bool, client InternalTestClient) {
	ctx, store := testSetup(client)
	podFoo := &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}
	podBar := &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "bar"}}

	tests := []struct {
		name       string
		key        string
		pred       storage.SelectionPredicate
		watchTests []*testWatchStruct
	}{{
		name:       "create a key",
		key:        "/somekey-1",
		watchTests: []*testWatchStruct{{podFoo, true, watch.Added}},
		pred:       storage.Everything,
	}, {
		name:       "key updated to match predicate",
		key:        "/somekey-3",
		watchTests: []*testWatchStruct{{podFoo, false, ""}, {podBar, true, watch.Added}},
		pred: storage.SelectionPredicate{
			Label: labels.Everything(),
			Field: fields.ParseSelectorOrDie("metadata.name=bar"),
			GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
				pod := obj.(*example.Pod)
				return nil, fields.Set{"metadata.name": pod.Name}, nil
			},
		},
	}, {
		name:       "update",
		key:        "/somekey-4",
		watchTests: []*testWatchStruct{{podFoo, true, watch.Added}, {podBar, true, watch.Modified}},
		pred:       storage.Everything,
	}, {
		name:       "delete because of being filtered",
		key:        "/somekey-5",
		watchTests: []*testWatchStruct{{podFoo, true, watch.Added}, {podBar, true, watch.Deleted}},
		pred: storage.SelectionPredicate{
			Label: labels.Everything(),
			Field: fields.ParseSelectorOrDie("metadata.name!=bar"),
			GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
				pod := obj.(*example.Pod)
				return nil, fields.Set{"metadata.name": pod.Name}, nil
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := store.Watch(ctx, tt.key, storage.ListOptions{ResourceVersion: "0", Predicate: tt.pred, Recursive: recursive})
			if err != nil {
				t.Fatalf("Watch failed: %v", err)
			}
			var prevObj *example.Pod
			for _, watchTest := range tt.watchTests {
				out := &example.Pod{}
				key := tt.key
				if recursive {
					key = key + "/item"
				}
				err := store.GuaranteedUpdate(ctx, key, out, true, nil, storage.SimpleUpdate(
					func(runtime.Object) (runtime.Object, error) {
						return watchTest.obj, nil
					}), nil)
				if err != nil {
					t.Fatalf("GuaranteedUpdate failed: %v", err)
				}
				if watchTest.expectEvent {
					expectObj := out
					if watchTest.watchType == watch.Deleted {
						expectObj = prevObj
						expectObj.ResourceVersion = out.ResourceVersion
					}
					testCheckResult(t, watchTest.watchType, w, expectObj)
				}
				prevObj = out
			}
			w.Stop()
			testCheckStop(t, w)
		})
	}
}

func RunTestDeleteTriggerWatch(t *testing.T, client InternalTestClient) {
	ctx, store := testSetup(client)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	w, err := store.Watch(ctx, key, storage.ListOptions{ResourceVersion: storedObj.ResourceVersion, Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	if err := store.Delete(ctx, key, &example.Pod{}, nil, storage.ValidateAllObjectFunc, nil); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	testCheckEventType(t, watch.Deleted, w)
}

// TestWatchFromZero tests that
// - watch from 0 should sync up and grab the object added before
// - watch from 0 is able to return events for objects whose previous version has been compacted
func RunTestWatchFromZero(t *testing.T, client InternalTestClient) {
	ctx, store := testSetup(client)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns"}})

	w, err := store.Watch(ctx, key, storage.ListOptions{ResourceVersion: "0", Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckResult(t, watch.Added, w, storedObj)
	w.Stop()

	// Update
	out := &example.Pod{}
	err = store.GuaranteedUpdate(ctx, key, out, true, nil, storage.SimpleUpdate(
		func(runtime.Object) (runtime.Object, error) {
			return &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns", Annotations: map[string]string{"a": "1"}}}, nil
		}), nil)
	if err != nil {
		t.Fatalf("GuaranteedUpdate failed: %v", err)
	}

	// Make sure when we watch from 0 we receive an ADDED event
	w, err = store.Watch(ctx, key, storage.ListOptions{ResourceVersion: "0", Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckResult(t, watch.Added, w, out)
	w.Stop()

	// Update again
	out = &example.Pod{}
	err = store.GuaranteedUpdate(ctx, key, out, true, nil, storage.SimpleUpdate(
		func(runtime.Object) (runtime.Object, error) {
			return &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns"}}, nil
		}), nil)
	if err != nil {
		t.Fatalf("GuaranteedUpdate failed: %v", err)
	}

	// Compact previous versions
	revToCompact, err := store.versioner.ParseResourceVersion(out.ResourceVersion)
	if err != nil {
		t.Fatalf("Error converting %q to an int: %v", storedObj.ResourceVersion, err)
	}
	err = client.Compact(ctx, int64(revToCompact))
	if err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	// Make sure we can still watch from 0 and receive an ADDED event
	w, err = store.Watch(ctx, key, storage.ListOptions{ResourceVersion: "0", Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckResult(t, watch.Added, w, out)
}

// TestWatchFromNoneZero tests that
// - watch from non-0 should just watch changes after given version
func RunTestWatchFromNoneZero(t *testing.T, client InternalTestClient) {
	ctx, store := testSetup(client)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})

	w, err := store.Watch(ctx, key, storage.ListOptions{ResourceVersion: storedObj.ResourceVersion, Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	out := &example.Pod{}
	store.GuaranteedUpdate(ctx, key, out, true, nil, storage.SimpleUpdate(
		func(runtime.Object) (runtime.Object, error) {
			return &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "bar"}}, err
		}), nil)
	testCheckResult(t, watch.Modified, w, out)
}

func RunTestWatchError(t *testing.T, client InternalTestClient) {
	// this codec fails on decodes, which will bubble up so we can verify the behavior
	invalidCodec := &testCodec{apitesting.TestCodec(codecs, examplev1.SchemeGroupVersion)}
	invalidStore := newStore(client, invalidCodec, newPod, "", schema.GroupResource{Resource: "pods"}, &prefixTransformer{prefix: []byte("test!")}, true)
	ctx := genericapirequest.WithCluster(context.Background(), genericapirequest.Cluster{Name: logicalcluster.New("root")})
	w, err := invalidStore.Watch(ctx, "/abc", storage.ListOptions{ResourceVersion: "0", Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	codec := apitesting.TestCodec(codecs, examplev1.SchemeGroupVersion)
	validStore := newStore(client, codec, newPod, "", schema.GroupResource{Resource: "pods"}, &prefixTransformer{prefix: []byte("test!")}, true)
	if err := validStore.GuaranteedUpdate(ctx, "/abc", &example.Pod{}, true, nil, storage.SimpleUpdate(
		func(runtime.Object) (runtime.Object, error) {
			return &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}, nil
		}), nil); err != nil {
		t.Fatalf("GuaranteedUpdate failed: %v", err)
	}
	testCheckEventType(t, watch.Error, w)
}

func RunTestWatchContextCancel(t *testing.T, client InternalTestClient) {
	ctx, store := testSetup(client)
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	// When we watch with a canceled context, we should detect that it's context canceled.
	// We won't take it as error and also close the watcher.
	w, err := store.watcher.Watch(canceledCtx, "/abc", 0, false, false, storage.Everything)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case _, ok := <-w.ResultChan():
		if ok {
			t.Error("ResultChan() should be closed")
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("timeout after %v", wait.ForeverTestTimeout)
	}
}

func RunTestWatchErrResultNotBlockAfterCancel(t *testing.T, client InternalTestClient) {
	origCtx, store := testSetup(client)
	ctx, cancel := context.WithCancel(origCtx)
	w := store.watcher.createWatchChan(ctx, "/abc", 0, false, genericapirequest.Shard(""), &genericapirequest.Cluster{}, false, storage.Everything)
	// make resutlChan and errChan blocking to ensure ordering.
	w.resultChan = make(chan watch.Event)
	w.errChan = make(chan error)
	// The event flow goes like:
	// - first we send an error, it should block on resultChan.
	// - Then we cancel ctx. The blocking on resultChan should be freed up
	//   and run() goroutine should return.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		w.run()
		wg.Done()
	}()
	w.errChan <- fmt.Errorf("some error")
	cancel()
	wg.Wait()
}

func RunTestWatchDeleteEventObjectHaveLatestRV(t *testing.T, client InternalTestClient) {
	ctx, store := testSetup(client)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})

	w, err := store.Watch(ctx, key, storage.ListOptions{ResourceVersion: storedObj.ResourceVersion, Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	rv, err := APIObjectVersioner{}.ObjectResourceVersion(storedObj)
	if err != nil {
		t.Fatalf("failed to parse resourceVersion on stored object: %v", err)
	}
	etcdW := client.Watch(ctx, "/", true, false, int64(rv))

	if err := store.Delete(ctx, key, &example.Pod{}, &storage.Preconditions{}, storage.ValidateAllObjectFunc, nil); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	var e watch.Event
	watchCtx, _ := context.WithTimeout(ctx, wait.ForeverTestTimeout)
	select {
	case e = <-w.ResultChan():
	case <-watchCtx.Done():
		t.Fatalf("timed out waiting for watch event")
	}
	deletedRV, err := deletedRevision(watchCtx, etcdW)
	if err != nil {
		t.Fatalf("did not see delete event in raw watch: %v", err)
	}
	watchedDeleteObj := e.Object.(*example.Pod)

	watchedDeleteRev, err := store.versioner.ParseResourceVersion(watchedDeleteObj.ResourceVersion)
	if err != nil {
		t.Fatalf("ParseWatchResourceVersion failed: %v", err)
	}
	if int64(watchedDeleteRev) != deletedRV {
		t.Errorf("Object from delete event have version: %v, should be the same as etcd delete's mod rev: %d",
			watchedDeleteRev, deletedRV)
	}
}

func deletedRevision(ctx context.Context, watch <-chan *WatchResponse) (int64, error) {
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case wres := <-watch:
			for _, evt := range wres.Events {
				if evt.Type == EventTypeDelete && evt.Kv != nil {
					return evt.Kv.ModRevision, nil
				}
			}
		}
	}
}

func RunTestWatchInitializationSignal(t *testing.T, client InternalTestClient) {
	_, store := testSetup(client)

	ctx, _ := context.WithTimeout(genericapirequest.WithCluster(context.Background(), genericapirequest.Cluster{Name: logicalcluster.New("root")}), 5*time.Second)
	initSignal := utilflowcontrol.NewInitializationSignal()
	ctx = utilflowcontrol.WithInitializationSignal(ctx, initSignal)

	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	_, err := store.Watch(ctx, key, storage.ListOptions{ResourceVersion: storedObj.ResourceVersion, Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	initSignal.Wait()
}

func RunTestProgressNotify(t *testing.T, client InternalTestClient) {
	codec := apitesting.TestCodec(codecs, examplev1.SchemeGroupVersion)
	store := newStore(client, codec, newPod, "", schema.GroupResource{Resource: "pods"}, &prefixTransformer{prefix: []byte(defaultTestPrefix)}, false)
	ctx := genericapirequest.WithCluster(context.Background(), genericapirequest.Cluster{Name: logicalcluster.New("root")})

	key := "/somekey"
	input := &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "name"}}
	out := &example.Pod{}
	if err := store.Create(ctx, key, input, out, 0); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	opts := storage.ListOptions{
		ResourceVersion: out.ResourceVersion,
		Predicate:       storage.Everything,
		ProgressNotify:  true,
	}
	w, err := store.Watch(ctx, key, opts)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckResultFunc(t, watch.Bookmark, w, func(object runtime.Object) error {
		pod, ok := object.(*example.Pod)
		if !ok {
			return fmt.Errorf("got %T, not *example.Pod", object)
		}
		return resourceVersionNoLaterThan(out.ResourceVersion)(pod.ResourceVersion)
	})
}

type testWatchStruct struct {
	obj         *example.Pod
	expectEvent bool
	watchType   watch.EventType
}

type testCodec struct {
	runtime.Codec
}

func (c *testCodec) Decode(data []byte, defaults *schema.GroupVersionKind, into runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	return nil, nil, errTestingDecode
}

func testCheckEventType(t *testing.T, expectEventType watch.EventType, w watch.Interface) {
	select {
	case res := <-w.ResultChan():
		if res.Type != expectEventType {
			t.Errorf("event type want=%v, get=%v", expectEventType, res.Type)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("time out after waiting %v on ResultChan", wait.ForeverTestTimeout)
	}
}

func resourceVersionNoLaterThan(sentinel string) func(string) error {
	return func(resourceVersion string) error {
		objectVersioner := APIObjectVersioner{}
		actualRV, err := objectVersioner.ParseResourceVersion(resourceVersion)
		if err != nil {
			return err
		}
		expectedRV, err := objectVersioner.ParseResourceVersion(sentinel)
		if err != nil {
			return err
		}
		if actualRV < expectedRV {
			return fmt.Errorf("expected a resourceVersion no smaller than than %d, but got %d", expectedRV, actualRV)
		}
		return nil
	}
}

func testCheckResult(t *testing.T, expectEventType watch.EventType, w watch.Interface, expectObj *example.Pod) {
	testCheckResultFunc(t, expectEventType, w, func(object runtime.Object) error {
		expectNoDiff(t, "incorrect object", expectObj, object)
		return nil
	})
}

func testCheckResultFunc(t *testing.T, expectEventType watch.EventType, w watch.Interface, check func(object runtime.Object) error) {
	select {
	case res := <-w.ResultChan():
		if res.Type != expectEventType {
			t.Errorf("event type want=%v, get=%v", expectEventType, res.Type)
			return
		}
		if err := check(res.Object); err != nil {
			t.Error(err)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("time out after waiting %v on ResultChan", wait.ForeverTestTimeout)
	}
}

func testCheckStop(t *testing.T, w watch.Interface) {
	select {
	case e, ok := <-w.ResultChan():
		if ok {
			var obj string
			switch e.Object.(type) {
			case *example.Pod:
				obj = e.Object.(*example.Pod).Name
			case *metav1.Status:
				obj = e.Object.(*metav1.Status).Message
			}
			t.Errorf("ResultChan should have been closed. Event: %s. Object: %s", e.Type, obj)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("time out after waiting 1s on ResultChan")
	}
}
