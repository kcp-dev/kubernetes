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
)

type event struct {
	key              string
	value            []byte
	prevValue        []byte
	rev              int64
	isDeleted        bool
	isCreated        bool
	isProgressNotify bool
}

// parseData converts a KeyValue retrieved from an initial sync() listing to a synthetic isCreated event.
func parseData(key string, data []byte, resourceVersion int64) *event {
	return &event{
		key:       key,
		value:     data,
		prevValue: nil,
		rev:       resourceVersion,
		isDeleted: false,
		isCreated: true,
	}
}

func parseEvent(key string, e *changefeedEvent, resourceVersion int64) (*event, error) {
	ret := &event{
		key:       key,
		rev:       resourceVersion,
		isDeleted: e.After == nil,
		isCreated: e.Before == nil,
	}
	if e.After != nil {
		value, err := hex.DecodeString(strings.TrimPrefix(e.After.Value, "\\x"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode value: %w", err)
		}
		ret.value = value
	}
	if e.Before != nil {
		value, err := hex.DecodeString(strings.TrimPrefix(e.Before.Value, "\\x"))
		if err != nil {
			return nil, fmt.Errorf("failed to decode value: %w", err)
		}
		ret.prevValue = value
	}
	return ret, nil
}

func progressNotifyEvent(rev int64) *event {
	return &event{
		rev:              rev,
		isProgressNotify: true,
	}
}
