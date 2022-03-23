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
	"fmt"

	"github.com/cockroachdb/apd"
)

const (
	// datum is a time before anyone would have ever used this code, such that we can subtract it from
	// the timestamps in the CockroachDB Hybrid-Logical Clock to reduce the overall amount of data that
	// needs to be encoded, allowing us to pack more bits of the logical part of the clock into the
	// resourceVersion. This is equivalent to the date: 2022-01-01 00:00:00 +0000 UTC
	datum = int64(1640995200000000000)
	// numBits is the number of bits which we will use to pack logical clock values into. This operation
	// is only valid as long as the current time is small enough that the difference between it and the
	// datum leaves this many unused bits inside an int64. For 4 bits, this is valid until:
	// 2040-04-07 23:59:12.303423488 +0000 UTC
	numBits = 4
	// totalBits is the number of bits we have to work with before our int64 overflows
	totalBits = 63
)

// toResourceVersion converts a CockroachDB Hybrid-Logical Clock (expressed as a fixed-precision decimal)
// to an etcd-style logical clock (expressed as a signed 64-bit integer). The CRDB HLC is composed of two
// parts: a 64-bit unsigned integer that is the nanosecond timestamp in Unix time, and a 32-bit unsigned
// integer formatted into 10 digits past the decimal of a logical counter to deduplicate events happening
// in the same nanosecond. For example: 1646149211071227687.0000000001
//
// Notably, the etcd API and many consumers of k8s expect to parse a *signed* 64-bit integer for the
// logical clock and resourceVersion, respectively, even though these values may never be <1. Therefore,
// the nanosecond timestamp takes up almost all the available space in uint64.
//
// We know that nobody will use this program before 2022 (as it's already past Jan 1st), so we choose to
// exploit this to minimize the amount of data we need to store in the timestamp and maximize how much we
// can pack into the signed 64-bit integer expected from a resourceVersion. If we subtract the datum
// specified above, we have four bits of free space until 2040. CRDB developers said that they have not
// seen a logical clock higher than 4 in the wild, anecdotally, and we have not been able to simulate a
// higher level of contention either, even with a single-node cluster writing data only to memory.
//
// Therefore, while there is a finite window of time for which packing bits into this number is appropriate,
// it seems robust enough to proceed. Such packing also incurs no data loss, allowing round-tripping.
func toResourceVersion(hybridLogicalTimestamp *apd.Decimal) (int64, error) {
	var integral, fractional apd.Decimal
	hybridLogicalTimestamp.Modf(&integral, &fractional)

	timestamp, err := integral.Int64()
	if err != nil {
		return 0, fmt.Errorf("failed to convert timestamp to integer: %v", err) // should never happen
	}
	timestamp -= datum
	if timestamp >= 1<<(totalBits-numBits) {
		return 0, fmt.Errorf("nanosecond timestamp in HLC too large to shift: %s (is it past 2040?)", hybridLogicalTimestamp.String())
	}
	timestamp = timestamp << numBits

	if fractional.IsZero() {
		// there is no logical portion to this clock
		return timestamp, nil
	}

	var logicalTimestamp apd.Decimal
	multiplier := apd.New(1, 10)
	condition, err := apd.BaseContext.Mul(&logicalTimestamp, &fractional, multiplier)
	if err != nil {
		return 0, fmt.Errorf("failed to determine timestamp of logical clock: %v", err)
	}
	if _, err := condition.GoError(apd.DefaultTraps); err != nil {
		return 0, fmt.Errorf("failed to determine timestamp of logical clock: %v", err)
	}

	counter, err := logicalTimestamp.Int64()
	if err != nil {
		return 0, fmt.Errorf("failed to convert logical timestamp to counter: %v", err)
	}

	if counter >= 1<<numBits {
		return 0, fmt.Errorf("logical clock in HLC too large to shift into timestamp: %s", hybridLogicalTimestamp.String())
	}

	return timestamp + counter, nil
}

// toHybridLogicalClock is the inverse of toResourceVersion
func toHybridLogicalClock(resourceVersion int64) (*apd.Decimal, error) {
	timestamp := resourceVersion>>numBits + datum
	logical := apd.New(timestamp, 0)

	counter := resourceVersion & ((1 << numBits) - 1) // mask out the lower bits
	if counter == 0 {
		return logical, nil
	}
	fractional := apd.New(counter, -10)

	var hybridLogicalTimestamp apd.Decimal
	condition, err := apd.BaseContext.Add(&hybridLogicalTimestamp, logical, fractional)
	if err != nil {
		return nil, fmt.Errorf("failed to determine timestamp of hybrid logical clock: %v", err)
	}
	if _, err := condition.GoError(apd.DefaultTraps); err != nil {
		return nil, fmt.Errorf("failed to determine timestamp of hybrid logical clock: %v", err)
	}
	return &hybridLogicalTimestamp, nil
}