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
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/cockroachdb/apd"
)

type eventType int

const (
	eventTypeProgressNotify eventType = iota
	eventTypeCreated
	eventTypeUpdated
	eventTypeDeleted
	eventTypeCompacted
)

type event struct {
	key       string
	cluster   string
	value     []byte
	prevValue []byte
	rev       int64
	rawRev    *apd.Decimal
	eType     eventType
}

// parseData converts a KeyValue retrieved from an initial sync() listing to a synthetic isCreated event.
func parseData(key string, data []byte, resourceVersion int64) *event {
	return &event{
		key:       key,
		value:     data,
		prevValue: nil,
		rev:       resourceVersion,
		eType:     eventTypeCreated,
	}
}

func parseEvent(e *changefeedEvent) (*event, error) {
	var rawRev *apd.Decimal
	if e.Resolved != nil {
		rawRev = e.Resolved
	} else {
		rawRev = e.MVCCTimestamp
	}
	resourceVersion, err := toResourceVersion(rawRev)
	if err != nil {
		return nil, err
	}
	ret := &event{
		rev:    resourceVersion,
		rawRev: rawRev,
	}
	switch {
	case e.Resolved != nil:
		ret.eType = eventTypeProgressNotify
	case e.After == nil:
		ret.eType = eventTypeDeleted
	case e.Before == nil:
		ret.eType = eventTypeCreated
	default:
		ret.eType = eventTypeUpdated
	}
	if e.After != nil {
		ret.key = e.After.Key
		ret.cluster = e.After.Cluster
		value, err := hex.DecodeString(strings.TrimPrefix(e.After.Value, "\\x"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode value: %w", err)
		}
		ret.value = value
	}
	if e.Before != nil {
		ret.key = e.Before.Key
		ret.cluster = e.Before.Cluster
		value, err := hex.DecodeString(strings.TrimPrefix(e.Before.Value, "\\x"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode value: %w", err)
		}
		ret.prevValue = value
	}
	return ret, nil
}
