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
	"strings"
	"time"

	"github.com/cockroachdb/apd"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/storage/generic"
)

func NewTestClient(ctx context.Context, c pool) generic.TestClient {
	return &testClient{
		client: NewClient(ctx, false, c),
	}
}

type testClient struct {
	*client
}

func (c *testClient) Compact(ctx context.Context, revision int64) error {
	// TODO: is there something better than this? no obvious way to trigger a compaction, and definitely not for a specific revision.
	// TODO: maybe restore from backup?
	hybridLogcialTimestamp, err := toHybridLogicalClock(revision)
	if err != nil {
		return fmt.Errorf("failed to determine hybrid logical clock when handling compaction event: %w", err)
	}
	revisionTimestampNanoseconds, err := hybridLogcialTimestamp.Int64()
	if err != nil && !strings.Contains(err.Error(), "has fractional part") {
		// having a fractional part means we have a logical clock, but that only disambiguates
		// between ns, so it's ok to ignore that error
		return err
	}
	revisionTimestamp := time.Unix(0, revisionTimestampNanoseconds)
	// incoming revision will be a ns timestamp, so why don't we fetch the current time and
	// figure out how long the GC interval would have to be so we would no longer hold the
	// incoming revision (as of now)
	var clusterHybridLogicalTimestamp apd.Decimal
	if err := c.pool.QueryRow(ctx, `SELECT cluster_logical_timestamp();`).Scan(&clusterHybridLogicalTimestamp); err != nil {
		return err
	}
	clusterTimestampNanoseconds, err := clusterHybridLogicalTimestamp.Int64()
	if err != nil && !strings.Contains(err.Error(), "has fractional part") {
		// having a fractional part means we have a logical clock, but that only disambiguates
		// between ns, so it's ok to ignore that error
		return err
	}
	clusterTimestamp := time.Unix(0, clusterTimestampNanoseconds)

	age := clusterTimestamp.Sub(revisionTimestamp)
	fmt.Printf("COMPACTION: %s - %s = %s\n", clusterTimestamp, revisionTimestamp, age)
	if age < time.Second {
		age = time.Second
	}
	if _, err := c.pool.Exec(ctx, `ALTER TABLE k8s CONFIGURE ZONE USING gc.ttlseconds = $1;`, age.Seconds()); err != nil {
		return err
	}

	// in order to wait for the compaction, ask for the revision and wait until it's out of scope
	return wait.Poll(500*time.Millisecond, 10*age, func() (done bool, err error) {
		var key string
		if err := c.QueryRow(ctx, fmt.Sprintf(`SELECT key FROM k8s AS OF SYSTEM TIME %s LIMIT 1;`, hybridLogcialTimestamp)).Scan(&key); err != nil {
			if isGarbageCollectionError(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}

var _ generic.TestClient = &testClient{}