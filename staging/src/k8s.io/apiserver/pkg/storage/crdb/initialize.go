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
	"time"
)

func InitializeDB(ctx context.Context, client pool, compactionInterval time.Duration) error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS k8s
			(
				key VARCHAR(512) NOT NULL PRIMARY KEY,
				value BLOB NOT NULL
			);`,
		// enable watches
		`SET CLUSTER SETTING kv.rangefeed.enabled = true;`,
		// set the latency floor for events
		`SET CLUSTER SETTING changefeed.experimental_poll_interval = '0.2s';`,
	} {
		if _, err := client.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("error initializing the database: %w", err)
		}
	}

	if compactionInterval != 0 {
		if _, err := client.Exec(ctx, `ALTER TABLE k8s CONFIGURE ZONE USING gc.ttlseconds = $1;`, compactionInterval.Round(time.Second).Seconds()); err != nil {
			return fmt.Errorf("failed to configure compaction interval: %w", err)
		}
	}

	return nil
}
