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

package etcd3

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/apitesting"
	"k8s.io/apimachinery/pkg/util/json"
	examplev1 "k8s.io/apiserver/pkg/apis/example/v1"
	"k8s.io/apiserver/pkg/storage/crdb"
	"k8s.io/apiserver/pkg/storage/versioner"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/apis/example"
	"k8s.io/apiserver/pkg/storage"
	utilflowcontrol "k8s.io/apiserver/pkg/util/flowcontrol"
)

func TestWatch(t *testing.T) {
	testWatch(t, false)
}

func TestWatchList(t *testing.T) {
	testWatch(t, true)
}

// It tests that
// - first occurrence of objects should notify Add event
// - update should trigger Modified event
// - update that gets filtered should trigger Deleted event
func testWatch(t *testing.T, recursive bool) {
	runTestWatch(t, TestBoostrapper(nil), recursive)
}

func TestWatchCRDB(t *testing.T) {
	testWatchCRDB(t, false)
}

func TestWatchListCRDB(t *testing.T) {
	testWatchCRDB(t, true)
}

func testWatchCRDB(t *testing.T, recursive bool) {
	runTestWatch(t, crdb.TestBootstrapper(), recursive)
}

func runTestWatch(t *testing.T, bootstrapper storage.TestBootstrapper, recursive bool) {
	ctx, store, _ := setup(t, bootstrapper)
	podFoo := &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}
	podBar := &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "bar"}}

	tests := []struct {
		name string
		key        string
		pred       storage.SelectionPredicate
		watchTests []*testWatchStruct
	}{{
		name: "create a key",
		key:        "/somekey-1",
		watchTests: []*testWatchStruct{{podFoo, true, watch.Added}},
		pred:       storage.Everything,
	}, {
		name: "create a key but obj gets filtered. Then update it with unfiltered obj",
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
		name: "update",
		key:        "/somekey-4",
		watchTests: []*testWatchStruct{{podFoo, true, watch.Added}, {podBar, true, watch.Modified}},
		pred:       storage.Everything,
	}, {
		name: "delete because of being filtered",
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
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w watch.Interface
			var err error
			if recursive {
				w, err = store.WatchList(ctx, tt.key, storage.ListOptions{ResourceVersion: "0", Predicate: tt.pred})
			} else {
				w, err = store.Watch(ctx, tt.key, storage.ListOptions{ResourceVersion: "0", Predicate: tt.pred})
			}
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
					testCheckResult(t, i, watchTest.watchType, w, expectObj)
				}
				prevObj = out
			}
			w.Stop()
			testCheckStop(t, i, w)
		})
	}
}

func TestDeleteTriggerWatch(t *testing.T) {
	RunTestDeleteTriggerWatch(t, TestBoostrapper(nil))
}

func TestDeleteTriggerWatchCRDB(t *testing.T) {
	RunTestDeleteTriggerWatch(t, crdb.TestBootstrapper())
}

func RunTestDeleteTriggerWatch(t *testing.T, bootstrapper storage.TestBootstrapper) {
	ctx, store, _ := setup(t, bootstrapper)
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

func TestWatchFromZero(t *testing.T) {
	RunTestWatchFromZero(t, TestBoostrapper(nil))
}

func TestWatchFromZeroCRDB(t *testing.T) {
	RunTestWatchFromZero(t, crdb.TestBootstrapper())
}
// TestWatchFromZero tests that
// - watch from 0 should sync up and grab the object added before
// - watch from 0 is able to return events for objects whose previous version has been compacted
func RunTestWatchFromZero(t *testing.T, bootstrapper storage.TestBootstrapper) {
	ctx, store, client := setup(t, bootstrapper)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "ns"}})

	w, err := store.Watch(ctx, key, storage.ListOptions{ResourceVersion: "0", Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckResult(t, 0, watch.Added, w, storedObj)
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
	testCheckResult(t, 1, watch.Added, w, out)
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
	revToCompact, err := versioner.APIObjectVersioner{}.ParseResourceVersion(out.ResourceVersion)
	if err != nil {
		t.Fatalf("Error converting %q to an int: %v", storedObj.ResourceVersion, err)
	}

	if err = client.RawCompact(ctx, int64(revToCompact)); err != nil {
		t.Fatalf("Error compacting: %v", err)
	}

	// Make sure we can still watch from 0 and receive an ADDED event
	w, err = store.Watch(ctx, key, storage.ListOptions{ResourceVersion: "0", Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckResult(t, 2, watch.Added, w, out)
}

func TestWatchFromNoneZero(t *testing.T) {
	RunTestWatchFromNoneZero(t, TestBoostrapper(nil))
}

func TestWatchFromNoneZeroCRDB(t *testing.T) {
	RunTestWatchFromNoneZero(t, crdb.TestBootstrapper())
}

// TestWatchFromNoneZero tests that
// - watch from non-0 should just watch changes after given version
func RunTestWatchFromNoneZero(t *testing.T, bootstrapper storage.TestBootstrapper) {
	ctx, store, _ := setup(t, bootstrapper)
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
	testCheckResult(t, 0, watch.Modified, w, out)
}

func TestWatchError(t *testing.T) {
	RunTestWatchError(t, TestBoostrapper(nil))
}

func TestWatchErrorCRDB(t *testing.T) {
	RunTestWatchError(t, crdb.TestBootstrapper())
}

func RunTestWatchError(t *testing.T, bootstrapper storage.TestBootstrapper) {
	ctx, invalidStore, client := setup(t, bootstrapper, withCodec(&testCodec{apitesting.TestCodec(codecs, examplev1.SchemeGroupVersion)}))
	w, err := invalidStore.Watch(ctx, "/abc", storage.ListOptions{ResourceVersion: "0", Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	validStore := additionalStore(t, bootstrapper, client)
	err = validStore.GuaranteedUpdate(ctx, "/abc", &example.Pod{}, true, nil, storage.SimpleUpdate(
		func(runtime.Object) (runtime.Object, error) {
			return &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}, nil
		}), nil)
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	testCheckEventType(t, watch.Error, w)
}

func TestWatchContextCancel(t *testing.T) {
	RunTestWatchContextCancel(t, TestBoostrapper(nil))
}

func TestWatchContextCancelCRDB(t *testing.T) {
	RunTestWatchContextCancel(t, crdb.TestBootstrapper())
}

func RunTestWatchContextCancel(t *testing.T, bootstrapper storage.TestBootstrapper) {
	ctx, store, _ := setup(t, bootstrapper)
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	// When we watch with a canceled context, we should detect that it's context canceled.
	// We won't take it as error and also close the watcher.
	w, err := store.Watch(canceledCtx, "/abc", storage.ListOptions{Predicate: storage.Everything})
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

// TODO: this is very low-level ... we can just dupe it in our package?
func TestWatchErrResultNotBlockAfterCancel(t *testing.T) {
	origCtx, store, _ := testSetup(t)
	ctx, cancel := context.WithCancel(origCtx)
	w := store.watcher.createWatchChan(ctx, "/abc", 0, false, "", false, storage.Everything)
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

func TestWatchDeleteEventObjectHaveLatestRV(t *testing.T) {
	RunTestWatchDeleteEventObjectHaveLatestRV(t, TestBoostrapper(nil))
}

func TestWatchDeleteEventObjectHaveLatestRVCRDB(t *testing.T) {
	RunTestWatchDeleteEventObjectHaveLatestRV(t, crdb.TestBootstrapper())
}

func RunTestWatchDeleteEventObjectHaveLatestRV(t *testing.T, bootstrapper storage.TestBootstrapper) {
	ctx, store, client := setup(t, bootstrapper)
	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})

	w, err := store.Watch(ctx, key, storage.ListOptions{ResourceVersion: storedObj.ResourceVersion, Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	createdRev, err := versioner.APIObjectVersioner{}.ParseResourceVersion(storedObj.ResourceVersion)
	if err != nil {
		t.Fatalf("ParseWatchResourceVersion failed: %v", err)
	}
	etcdW := client.RawWatch(ctx, "/", int64(createdRev))

	if err := store.Delete(ctx, key, &example.Pod{}, &storage.Preconditions{}, storage.ValidateAllObjectFunc, nil); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	e := <-w.ResultChan()
	watchedDeleteObj := e.Object.(*example.Pod)
	wres := <-etcdW
	if wres.Error != nil {
		t.Fatalf("error from raw watch: %v", wres.Error)
	}

	watchedDeleteRev, err := versioner.APIObjectVersioner{}.ParseResourceVersion(watchedDeleteObj.ResourceVersion)
	if err != nil {
		t.Fatalf("ParseWatchResourceVersion failed: %v", err)
	}
	if int64(watchedDeleteRev) != wres.Revision {
		t.Errorf("Object from delete event have version: %v, should be the same as etcd delete's mod rev: %d",
			watchedDeleteRev, wres.Revision)
	}
}

func TestWatchInitializationSignal(t *testing.T) {
	RunTestWatchInitializationSignal(t, TestBoostrapper(nil))
}

func TestWatchInitializationSignalCRDB(t *testing.T) {
	RunTestWatchInitializationSignal(t, crdb.TestBootstrapper())
}

func RunTestWatchInitializationSignal(t *testing.T, bootstrapper storage.TestBootstrapper) {
	ctx, store, _ := setup(t, bootstrapper)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	t.Cleanup(cancel)
	initSignal := utilflowcontrol.NewInitializationSignal()
	ctx = utilflowcontrol.WithInitializationSignal(ctx, initSignal)

	key, storedObj := testPropogateStore(ctx, t, store, &example.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo"}})
	_, err := store.Watch(ctx, key, storage.ListOptions{ResourceVersion: storedObj.ResourceVersion, Predicate: storage.Everything})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	initSignal.Wait()
}

func TestProgressNotify(t *testing.T) {
	RunTestProgressNotify(t, TestBoostrapper(nil, WithProgressNotifyInterval(time.Second)))
}

func TestProgressNotifyCRDB(t *testing.T) {
	RunTestProgressNotify(t, crdb.TestBootstrapper())
}

func RunTestProgressNotify(t *testing.T, bootstrapper storage.TestBootstrapper) {
	ctx, store, _ := setup(t, bootstrapper)

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
	testCheckResultFunc(t, 0, watch.Bookmark, w, func(object runtime.Object) error {
		pod, ok := object.(*example.Pod)
		if !ok {
			return fmt.Errorf("got %T, not *example.Pod", object)
		}
		objectVersioner := versioner.APIObjectVersioner{}
		actualRV, err := objectVersioner.ParseResourceVersion(pod.ResourceVersion)
		if err != nil {
			return err
		}
		expectedRV, err := objectVersioner.ParseResourceVersion(out.ResourceVersion)
		if err != nil {
			return err
		}
		if actualRV < expectedRV {
			return fmt.Errorf("expected a resourceVersion in the bookmark larger than %d, but got %d", expectedRV, actualRV)
		}
		return nil
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

func testCheckResult(t *testing.T, i int, expectEventType watch.EventType, w watch.Interface, expectObj *example.Pod) {
	testCheckResultFunc(t, i, expectEventType, w, func(object runtime.Object) error {
		if diff := cmp.Diff(expectObj, object); diff != "" {
			return fmt.Errorf("#%d: obj dif: %s", i, diff)
		}
		return nil
	})
}

func testCheckResultFunc(t *testing.T, i int, expectEventType watch.EventType, w watch.Interface, check func(object runtime.Object) error) {
	select {
	case res := <-w.ResultChan():
		raw, err := json.Marshal(res)
		if err != nil {
			t.Errorf("#%d: failed to marshal result: %v", i, err)
		}
		t.Log(string(raw))
		if res.Type != expectEventType {
			t.Errorf("#%d: event type want=%v, get=%v", i, expectEventType, res.Type)
			return
		}
		if err := check(res.Object); err != nil {
			t.Errorf("#%d: obj err: %v", i, err)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("#%d: time out after waiting %v on ResultChan", i, wait.ForeverTestTimeout)
	}
}

func testCheckStop(t *testing.T, i int, w watch.Interface) {
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
			t.Errorf("#%d: ResultChan should have been closed. Event: %s. Object: %s", i, e.Type, obj)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("#%d: time out after waiting 1s on ResultChan", i)
	}
}
