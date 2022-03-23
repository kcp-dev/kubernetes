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
	"testing"
	"time"

	"k8s.io/apiserver/pkg/storage/etcd3/testserver"
	"k8s.io/apiserver/pkg/storage/generic"
)

func TestWatch(t *testing.T) {
	generic.RunTestWatch(t, newTestClient(t))
}

func TestDeleteTriggerWatch(t *testing.T) {
	generic.RunTestDeleteTriggerWatch(t, newTestClient(t))
}

// TestWatchFromZero tests that
// - watch from 0 should sync up and grab the object added before
// - watch from 0 is able to return events for objects whose previous version has been compacted
func TestWatchFromZero(t *testing.T) {
	generic.RunTestWatchFromZero(t, newTestClient(t))
}

// TestWatchFromNoneZero tests that
// - watch from non-0 should just watch changes after given version
func TestWatchFromNoneZero(t *testing.T) {
	generic.RunTestWatchFromNoneZero(t, newTestClient(t))
}

func TestWatchError(t *testing.T) {
	generic.RunTestWatchError(t, newTestClient(t))
}

func TestWatchContextCancel(t *testing.T) {
	generic.RunTestWatchContextCancel(t, newTestClient(t))
}

func TestWatchErrResultNotBlockAfterCancel(t *testing.T) {
	generic.RunTestWatchErrResultNotBlockAfterCancel(t, newTestClient(t))
}

func TestWatchDeleteEventObjectHaveLatestRV(t *testing.T) {
	generic.RunTestWatchDeleteEventObjectHaveLatestRV(t, newTestClient(t))
}

func TestWatchInitializationSignal(t *testing.T) {
	generic.RunTestWatchInitializationSignal(t, newTestClient(t))
}

func TestProgressNotify(t *testing.T) {
	clusterConfig := testserver.NewTestConfig(t)
	clusterConfig.ExperimentalWatchProgressNotifyInterval = time.Second
	generic.RunTestProgressNotify(t, newTestClientCfg(t, clusterConfig))
}
